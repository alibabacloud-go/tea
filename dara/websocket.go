package dara

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

type WebSocketMessageType int

const (
	WebSocketMessageTypeText WebSocketMessageType = iota
	WebSocketMessageTypeBinary
	WebSocketMessageTypePing
	WebSocketMessageTypePong
	WebSocketMessageTypeClose
)

type WebSocketMessage struct {
	Type      WebSocketMessageType
	Payload   []byte
	Headers   map[string]string
	Timestamp time.Time
}

type WebSocketCloseFrame struct {
	Code   int
	Reason string
}

type WebSocketSessionInfo struct {
	SessionID   string // Session ID from server (x-acs-ws-session-id header)
	RequestID   string // Request ID from server (x-acs-request-id header)
	ConnectedAt time.Time
	RemoteAddr  string
	LocalAddr   string
	Attributes  map[string]interface{}
}

type WebSocketHandler interface {
	AfterConnectionEstablished(session *WebSocketSessionInfo) error
	HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error
	HandleError(session *WebSocketSessionInfo, err error) error
	AfterConnectionClosed(session *WebSocketSessionInfo, code int, reason string) error
	SupportsPartialMessages() bool
}

type WebSocketClient interface {
	Connect(ctx context.Context, request *Request, runtimeObject *RuntimeObject) (map[string]interface{}, error)
	Disconnect(ctx context.Context) error
	Reconnect(ctx context.Context) (map[string]interface{}, error)
	ReconnectGracefully(ctx context.Context) (map[string]interface{}, error) // Graceful reconnection with session ID
	IsConnected() bool
	SendText(ctx context.Context, text string) error
	SendBinary(ctx context.Context, data []byte) error
	GetSessionInfo() *WebSocketSessionInfo
	Close() error
}

type DefaultWebSocketClient struct {
	handler      WebSocketHandler // 必需：消息处理器
	stopChan     chan struct{}    // 必需：控制 goroutine 停止
	pongReceived chan struct{}    // 必需：ping/pong 机制
	state        int32            // 必需：连接状态 (0=disconnected, 1=connecting, 2=connected, 3=disconnecting)

	conn          *websocket.Conn       // 连接时建立
	session       *WebSocketSessionInfo // 连接时创建
	request       *Request              // 连接时传入
	runtimeObject *RuntimeObject        // 连接时传入
	ctx           context.Context       // 连接时创建, 管理连接生命周期
	cancel        context.CancelFunc    // 连接时创建, 管理连接生命周期

	// 运行时赋值的字段
	reconnectCount int          // 重连时递增
	pingTicker     *time.Ticker // 启动 ping/pong 时创建
	closed         bool         // 关闭时设置

	// 超时配置（在 Connect 时一次性计算，后续直接使用）
	pingInterval      time.Duration // Ping 间隔
	reconnectInterval time.Duration // 重连间隔
	writeTimeout      time.Duration // 写入超时
	readTimeout       time.Duration // 读取超时
	pongTimeout       time.Duration // Pong 超时
	maxReconnectTimes int           // 最大重连次数

	// 零值可用的字段（sync 类型）
	reconnectMu sync.Mutex
	closeMu     sync.Mutex
	wg          sync.WaitGroup
}

func NewDefaultWebSocketClient(handler WebSocketHandler) (*DefaultWebSocketClient, error) {
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}

	client := &DefaultWebSocketClient{
		handler:      handler,
		stopChan:     make(chan struct{}),
		pongReceived: make(chan struct{}, 1),
		state:        0, // disconnected
	}

	return client, nil
}

func NewWebSocketClientAndConnect(request *Request, runtimeObject *RuntimeObject) (*DefaultWebSocketClient, map[string]interface{}, error) {
	if runtimeObject == nil {
		return nil, nil, errors.New("runtimeObject cannot be nil")
	}

	var handler WebSocketHandler
	if runtimeObject.WebSocketHandler != nil {
		if wsHandler, ok := runtimeObject.WebSocketHandler.(WebSocketHandler); ok {
			handler = wsHandler
		}
	}

	if handler == nil {
		return nil, nil, errors.New("WebSocketHandler is required: please set it in runtimeObject.WebSocketHandler")
	}

	ctx := context.Background()

	client, err := NewDefaultWebSocketClient(handler)
	if err != nil {
		return nil, nil, err
	}

	result, err := client.Connect(ctx, request, runtimeObject)
	if err != nil {
		return nil, nil, err
	}

	return client, result, nil
}

func buildWebSocketURL(request *Request) (string, error) {
	if request == nil {
		return "", errors.New("request cannot be nil")
	}

	if request.Protocol == nil {
		request.Protocol = String("ws")
	} else {
		protocol := strings.ToLower(StringValue(request.Protocol))
		if protocol == "http" {
			protocol = "ws"
		} else if protocol == "https" {
			protocol = "wss"
		}
		request.Protocol = String(protocol)
	}

	if request.Domain == nil {
		if host, ok := request.Headers["host"]; ok && host != nil {
			request.Domain = host
		} else {
			return "", errors.New("domain is required (set in request.Headers[\"host\"] or request.Domain)")
		}
	}

	requestURL := fmt.Sprintf("%s://%s", StringValue(request.Protocol), StringValue(request.Domain))

	if request.Pathname != nil {
		requestURL += StringValue(request.Pathname)
	} else {
		requestURL += "/"
	}

	if request.Query != nil && len(request.Query) > 0 {
		q := url.Values{}
		for key, value := range request.Query {
			if value != nil {
				q.Add(key, StringValue(value))
			}
		}
		querystring := q.Encode()
		if len(querystring) > 0 {
			if strings.Contains(requestURL, "?") {
				requestURL = fmt.Sprintf("%s&%s", requestURL, querystring)
			} else {
				requestURL = fmt.Sprintf("%s?%s", requestURL, querystring)
			}
		}
	}

	return requestURL, nil
}

func (c *DefaultWebSocketClient) updateTimeoutConfig(runtimeObject *RuntimeObject) {
	c.pingInterval = time.Duration(IntValue(runtimeObject.WebSocketPingInterval)) * time.Millisecond
	if c.pingInterval <= 0 {
		c.pingInterval = 0 // No ping if not configured
	}

	c.reconnectInterval = time.Duration(IntValue(runtimeObject.WebSocketReconnectInterval)) * time.Millisecond
	if c.reconnectInterval <= 0 {
		c.reconnectInterval = 5 * time.Second // Default
	}

	c.writeTimeout = time.Duration(IntValue(runtimeObject.WebSocketWriteTimeout)) * time.Millisecond
	if c.writeTimeout <= 0 {
		c.writeTimeout = 30 * time.Second // Default
	}

	c.readTimeout = time.Duration(IntValue(runtimeObject.ReadTimeout)) * time.Millisecond
	if c.readTimeout <= 0 {
		c.readTimeout = 0 // No read timeout if not configured
	}

	c.pongTimeout = time.Duration(IntValue(runtimeObject.WebSocketPongTimeout)) * time.Millisecond
	if c.pongTimeout <= 0 {
		c.pongTimeout = 10 * time.Second // Default
	}

	c.maxReconnectTimes = IntValue(runtimeObject.WebSocketMaxReconnectTimes)
	if c.maxReconnectTimes <= 0 {
		c.maxReconnectTimes = 5 // Default
	}
}

func (c *DefaultWebSocketClient) Connect(ctx context.Context, request *Request, runtimeObject *RuntimeObject) (map[string]interface{}, error) {
	if request == nil {
		return map[string]interface{}{"success": false, "error": "request cannot be nil"}, errors.New("request cannot be nil")
	}
	if runtimeObject == nil {
		runtimeObject = &RuntimeObject{}
	}

	c.request = request
	c.runtimeObject = runtimeObject

	c.updateTimeoutConfig(runtimeObject)

	atomic.StoreInt32(&c.state, 1) // connecting

	// Create a cancellable context for this connection
	// This allows us to cancel reconnect operations when the client is closed
	c.closeMu.Lock()
	if c.cancel != nil {
		c.cancel() // Cancel previous context if exists
	}
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.closeMu.Unlock()

	requestURL, err := buildWebSocketURL(request)
	if err != nil {
		atomic.StoreInt32(&c.state, 0) // disconnected
		return map[string]interface{}{"success": false, "error": err.Error()}, err
	}

	u, err := url.Parse(requestURL)
	if err != nil {
		atomic.StoreInt32(&c.state, 0) // disconnected
		return map[string]interface{}{"success": false, "error": err.Error()}, err
	}

	handshakeTimeout := time.Duration(IntValue(runtimeObject.WebSocketHandshakeTimeout)) * time.Millisecond
	if handshakeTimeout <= 0 {
		handshakeTimeout = 30 * time.Second // Default
	}
	connectTimeout := time.Duration(IntValue(runtimeObject.ConnectTimeout)) * time.Millisecond
	if connectTimeout <= 0 {
		connectTimeout = 10 * time.Second // Default
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}

	if u.Scheme == "wss" || u.Scheme == "https" {
		if BoolValue(runtimeObject.IgnoreSSL) != true {
			dialer.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: false,
			}
			// Client certificate and key
			if runtimeObject.Key != nil && runtimeObject.Cert != nil && StringValue(runtimeObject.Key) != "" && StringValue(runtimeObject.Cert) != "" {
				cert, err := tls.X509KeyPair([]byte(StringValue(runtimeObject.Cert)), []byte(StringValue(runtimeObject.Key)))
				if err != nil {
					atomic.StoreInt32(&c.state, 0) // disconnected
					return map[string]interface{}{"success": false, "error": fmt.Sprintf("failed to load client certificate: %v", err)}, err
				}
				dialer.TLSClientConfig.Certificates = []tls.Certificate{cert}
			}
			if runtimeObject.Ca != nil && StringValue(runtimeObject.Ca) != "" {
				clientCertPool := x509.NewCertPool()
				ok := clientCertPool.AppendCertsFromPEM([]byte(StringValue(runtimeObject.Ca)))
				if !ok {
					atomic.StoreInt32(&c.state, 0) // disconnected
					return map[string]interface{}{"success": false, "error": "failed to parse root certificate"}, errors.New("failed to parse root certificate")
				}
				dialer.TLSClientConfig.RootCAs = clientCertPool
			}
		} else {
			dialer.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}

	// Priority: SOCKS5 > HTTP/HTTPS proxy
	if runtimeObject.Socks5Proxy != nil && StringValue(runtimeObject.Socks5Proxy) != "" {
		socks5Proxy, err := url.Parse(StringValue(runtimeObject.Socks5Proxy))
		if err == nil {
			var auth *proxy.Auth
			if socks5Proxy.User != nil {
				password, _ := socks5Proxy.User.Password()
				auth = &proxy.Auth{
					User:     socks5Proxy.User.Username(),
					Password: password,
				}
			}
			socks5Network := strings.ToLower(StringValue(runtimeObject.Socks5NetWork))
			if socks5Network == "" {
				socks5Network = "tcp"
			}
			socks5Dialer, err := proxy.SOCKS5(socks5Network, socks5Proxy.Host, auth,
				&net.Dialer{
					Timeout:   connectTimeout,
					DualStack: true,
					LocalAddr: getLocalAddr(StringValue(runtimeObject.LocalAddr)),
				})
			if err != nil {
				atomic.StoreInt32(&c.state, 0) // disconnected
				return map[string]interface{}{"success": false, "error": fmt.Sprintf("failed to setup SOCKS5 proxy: %v", err)}, err
			}
			dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return socks5Dialer.Dial(network, addr)
			}
		}
	} else {
		var httpProxy *url.URL
		var err error
		protocol := u.Scheme
		host := u.Hostname()

		noProxyList := getNoProxy(protocol, runtimeObject)
		shouldUseProxy := true
		for _, noProxyHost := range noProxyList {
			if noProxyHost == host {
				shouldUseProxy = false
				break
			}
		}

		if shouldUseProxy {
			var proxyURL *string
			if protocol == "wss" || protocol == "https" {
				proxyURL = runtimeObject.HttpsProxy
			} else {
				proxyURL = runtimeObject.HttpProxy
			}
			if proxyURL != nil && StringValue(proxyURL) != "" {
				httpProxy, err = getHttpProxy(protocol, host, runtimeObject)
			}
		}

		if httpProxy != nil && err == nil {
			dialer.Proxy = http.ProxyURL(httpProxy)
			// Add Proxy-Authorization header if needed
			if httpProxy.User != nil {
				password, _ := httpProxy.User.Password()
				auth := httpProxy.User.Username() + ":" + password
				basic := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
				if request.Headers == nil {
					request.Headers = make(map[string]*string)
				}
				request.Headers["Proxy-Authorization"] = String(basic)
			}
		}

		if runtimeObject.LocalAddr != nil && StringValue(runtimeObject.LocalAddr) != "" {
			localAddr := getLocalAddr(StringValue(runtimeObject.LocalAddr))
			if localAddr != nil {
				dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					d := &net.Dialer{
						Timeout:   connectTimeout,
						DualStack: true,
						LocalAddr: localAddr,
					}
					return d.DialContext(ctx, network, addr)
				}
			}
		} else {
			dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				d := &net.Dialer{
					Timeout:   connectTimeout,
					DualStack: true,
				}
				return d.DialContext(ctx, network, addr)
			}
		}
	}

	header := http.Header{}
	if request.Headers != nil {
		for k, v := range request.Headers {
			if v != nil && k != "host" && k != "content-length" {
				header.Set(k, StringValue(v))
			}
		}
	}

	fmt.Printf("[WebSocket] Handshake headers:\n")
	for k, v := range header {
		fmt.Printf("  %s: %v\n", k, v)
	}

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	conn, resp, err := dialer.DialContext(connectCtx, requestURL, header)
	if err != nil {
		atomic.StoreInt32(&c.state, 0) // disconnected
		if resp != nil {
			fmt.Printf("[WebSocket] Handshake failed. Response status: %s\n", resp.Status)
			fmt.Printf("[WebSocket] Response headers:\n")
			for k, v := range resp.Header {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
		return map[string]interface{}{"success": false, "error": err.Error()}, err
	}

	c.conn = conn
	atomic.StoreInt32(&c.state, 2) // connected

	c.setupPongHandler()

	// HTTP headers are case-insensitive, Go's Header.Get() is case-insensitive
	sessionID := ""
	requestID := ""
	if resp != nil && resp.Header != nil {
		sessionID = resp.Header.Get("x-acs-ws-session-id")
		requestID = resp.Header.Get("x-acs-request-id")
	}

	if sessionID == "" {
		sessionID = generateSessionID()
	}

	c.session = &WebSocketSessionInfo{
		SessionID:   sessionID,
		RequestID:   requestID,
		ConnectedAt: time.Now(),
		RemoteAddr:  conn.RemoteAddr().String(),
		LocalAddr:   conn.LocalAddr().String(),
		Attributes:  make(map[string]interface{}),
	}

	c.startMessageHandlers()

	// Start ping/pong if configured
	if c.pingInterval > 0 {
		c.startPingPong()
	}

	if err := c.handler.AfterConnectionEstablished(c.session); err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}, err
	}

	result := map[string]interface{}{
		"success":   true,
		"status":    resp.StatusCode,
		"header":    resp.Header,
		"session":   c.session,
		"sessionId": c.session.SessionID,
		"requestId": c.session.RequestID,
	}

	return result, nil
}

func (c *DefaultWebSocketClient) Disconnect(ctx context.Context) error {
	return c.disconnect(1000, "Normal closure")
}

func (c *DefaultWebSocketClient) disconnect(code int, reason string) error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}

	atomic.StoreInt32(&c.state, 3) // disconnecting

	c.stopPingPong()

	// Signal goroutines to stop first (before closing connection)
	// This allows goroutines to exit gracefully
	select {
	case <-c.stopChan:
		// Already closed, don't close again
	default:
		close(c.stopChan)
	}

	// Close connection (this will cause ReadMessage to return error)
	if c.conn != nil {
		deadline := time.Now().Add(time.Second)
		c.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(code, reason),
			deadline)
		c.conn.Close()
		c.conn = nil
	}

	// Wait for all goroutines to finish (with timeout to avoid deadlock)
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-time.After(5 * time.Second):
		// Timeout - log warning but continue
		fmt.Printf("[WebSocket] Warning: timeout waiting for goroutines to finish\n")
	}

	if c.session != nil {
		c.handler.AfterConnectionClosed(c.session, code, reason)
	}

	atomic.StoreInt32(&c.state, 0) // disconnected
	c.closed = true

	// Cancel context to stop any ongoing reconnect operations
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	return nil
}

func (c *DefaultWebSocketClient) Reconnect(ctx context.Context) (map[string]interface{}, error) {
	return c.reconnectInternal(ctx, false)
}

// ReconnectGracefully performs a graceful reconnection (server-initiated via RECONNECT control message, with session ID)
func (c *DefaultWebSocketClient) ReconnectGracefully(ctx context.Context) (map[string]interface{}, error) {
	return c.reconnectInternal(ctx, true)
}

// reconnectInternal is the internal implementation for both normal and graceful reconnection
// graceful: true for graceful reconnection (uses session ID), false for normal reconnection (doesn't use session ID)
func (c *DefaultWebSocketClient) reconnectInternal(ctx context.Context, graceful bool) (map[string]interface{}, error) {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	// Check if already connected (avoid unnecessary reconnection)
	if c.IsConnected() {
		fmt.Printf("[WebSocket] Already connected, skipping reconnect\n")
		return map[string]interface{}{"success": true, "already_connected": true}, nil
	}

	// Check if context is already cancelled (client might be closing)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if c.runtimeObject == nil || !BoolValue(c.runtimeObject.WebSocketEnableReconnect) {
		return nil, errors.New("reconnect is disabled")
	}

	if c.reconnectCount >= c.maxReconnectTimes {
		return nil, fmt.Errorf("max reconnect times reached: %d", c.maxReconnectTimes)
	}

	// Save previous session ID before cleanup (only for graceful reconnection)
	previousSessionID := ""
	if graceful {
		if c.session != nil && c.session.SessionID != "" {
			previousSessionID = c.session.SessionID
		} else {
			return nil, errors.New("graceful reconnection requires existing session ID")
		}
	}

	// Clean up resources before reconnecting
	c.cleanupResources()

	c.closed = false
	c.stopChan = make(chan struct{})
	c.reconnectCount++

	// Use context-aware sleep to allow cancellation during reconnect interval
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(c.reconnectInterval):
		// Continue with reconnect
	}

	// Check context again before attempting connection
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if c.request == nil {
		return nil, errors.New("request is nil, cannot reconnect")
	}

	// For graceful reconnection, add previous session ID to request headers
	// For normal reconnection, don't add session ID (or remove it if exists)
	if graceful && previousSessionID != "" {
		if c.request.Headers == nil {
			c.request.Headers = make(map[string]*string)
		}
		c.request.Headers["X-Acs-Ws-Session-Id"] = String(previousSessionID)
		fmt.Printf("[WebSocket] Graceful reconnection with session ID: %s\n", previousSessionID)
	} else {
		// Normal reconnection: remove session ID header if exists (to start fresh)
		if c.request.Headers != nil {
			delete(c.request.Headers, "X-Acs-Ws-Session-Id")
		}
		fmt.Printf("[WebSocket] Normal reconnection (without session ID)\n")
	}

	result, err := c.Connect(ctx, c.request, c.runtimeObject)
	if err == nil {
		c.reconnectCount = 0 // Reset on success
	}

	return result, err
}

// cleanupResources cleans up all resources (ping/pong, goroutines, connection, session)
// This is used before reconnecting to ensure a clean state
func (c *DefaultWebSocketClient) cleanupResources() {
	atomic.StoreInt32(&c.state, 3) // disconnecting

	c.stopPingPong()

	// Signal goroutines to stop first (before closing connection)
	// This allows goroutines to exit gracefully
	select {
	case <-c.stopChan:
		// Already closed, don't close again
	default:
		close(c.stopChan)
	}

	// Close connection (this will cause ReadMessage to return error)
	if c.conn != nil {
		deadline := time.Now().Add(time.Second)
		c.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "Reconnecting"),
			deadline)
		c.conn.Close()
		c.conn = nil // Clear reference
	}

	// Wait for all goroutines to finish (with timeout to avoid deadlock)
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-time.After(5 * time.Second):
		// Timeout - log warning but continue
		fmt.Printf("[WebSocket] Warning: timeout waiting for goroutines to finish during cleanup\n")
	}

	// Clear session (will be recreated on successful reconnect)
	c.session = nil

	atomic.StoreInt32(&c.state, 0) // disconnected
}

func (c *DefaultWebSocketClient) IsConnected() bool {
	return atomic.LoadInt32(&c.state) == 2
}

func (c *DefaultWebSocketClient) SendText(ctx context.Context, text string) error {
	if !c.IsConnected() {
		return errors.New("not connected")
	}

	if c.writeTimeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}

	return c.conn.WriteMessage(websocket.TextMessage, []byte(text))
}

func (c *DefaultWebSocketClient) SendBinary(ctx context.Context, data []byte) error {
	if !c.IsConnected() {
		return errors.New("not connected")
	}

	if c.writeTimeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}

	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (c *DefaultWebSocketClient) GetSessionInfo() *WebSocketSessionInfo {
	return c.session
}

func (c *DefaultWebSocketClient) Close() error {
	return c.disconnect(1000, "Client closed")
}

func (c *DefaultWebSocketClient) startMessageHandlers() {
	// Read messages from WebSocket
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.readMessages()
	}()
}

func (c *DefaultWebSocketClient) readMessages() {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in readMessages: %v", r)
			if c.session != nil {
				c.handler.HandleError(c.session, err)
			}
		}
	}()

	for {
		// Check stopChan first
		select {
		case <-c.stopChan:
			return
		default:
		}

		if c.conn == nil {
			return
		}

		// Set read deadline
		if c.readTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
		}

		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			// Check if we should stop (connection might be closed)
			select {
			case <-c.stopChan:
				return
			default:
			}

			// Connection closed or other error
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				if c.session != nil {
					c.handler.HandleError(c.session, err)
				}
			}

			// Try to reconnect only if not stopping
			select {
			case <-c.stopChan:
				return
			default:
				if BoolValue(c.runtimeObject.WebSocketEnableReconnect) {
					// Use the connection's context so reconnect can be cancelled
					c.closeMu.Lock()
					reconnectCtx := c.ctx
					c.closeMu.Unlock()
					if reconnectCtx != nil && c.request != nil {
						go c.Reconnect(reconnectCtx)
					}
				}
			}
			return
		}

		msg := &WebSocketMessage{
			Type:      convertToWebSocketMessageType(messageType),
			Payload:   data,
			Headers:   make(map[string]string),
			Timestamp: time.Now(),
		}

		fmt.Printf("[WebSocket] Received message: type=%d, size=%d bytes\n", messageType, len(data))

		if c.session != nil {
			// Check for RECONNECT control message for AWAP protocol
			// This allows graceful reconnection triggered by server
			if messageType == websocket.TextMessage {
				if awapMsg, err := ParseAwapMessage(msg); err == nil {
					if awapMsg.Type == AwapMessageTypeReconnect {
						fmt.Printf("[WebSocket] Received RECONNECT control message, initiating graceful reconnection\n")
						// Trigger graceful reconnection in a goroutine to avoid blocking message reading
						go func() {
							c.closeMu.Lock()
							reconnectCtx := c.ctx
							c.closeMu.Unlock()
							if reconnectCtx != nil {
								if _, err := c.ReconnectGracefully(reconnectCtx); err != nil {
									fmt.Printf("[WebSocket] Graceful reconnection failed: %v\n", err)
									if c.session != nil {
										c.handler.HandleError(c.session, err)
									}
								}
							}
						}()
						return
					}
				}
			}

			if err := c.handler.HandleRawMessage(c.session, msg); err != nil {
				fmt.Printf("[WebSocket] HandleRawMessage error: %v\n", err)
				c.handler.HandleError(c.session, err)
			}
		} else {
			fmt.Printf("[WebSocket] Warning: session is nil, cannot handle message\n")
		}
	}
}

func (c *DefaultWebSocketClient) startPingPong() {
	if c.pingInterval <= 0 {
		return
	}
	c.pingTicker = time.NewTicker(c.pingInterval)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		for {
			select {
			case <-c.stopChan:
				return
			case <-c.pingTicker.C:
				if c.conn == nil {
					return
				}

				deadline := time.Now().Add(c.writeTimeout)
				if err := c.conn.WriteControl(websocket.PingMessage, []byte{}, deadline); err != nil {
					if c.session != nil {
						c.handler.HandleError(c.session, err)
					}
					return
				}

				select {
				case <-c.pongReceived:
					// Pong received
				case <-time.After(c.pongTimeout):
					// Pong timeout, try to reconnect
					if BoolValue(c.runtimeObject.WebSocketEnableReconnect) {
						// Use the connection's context so reconnect can be cancelled
						c.closeMu.Lock()
						reconnectCtx := c.ctx
						c.closeMu.Unlock()
						if reconnectCtx != nil {
							go c.Reconnect(reconnectCtx)
						}
					}
					return
				}
			}
		}
	}()
}

// This should be called immediately after connection is established to avoid race conditions
func (c *DefaultWebSocketClient) setupPongHandler() {
	if c.conn == nil {
		return
	}
	c.conn.SetPongHandler(func(appData string) error {
		select {
		case c.pongReceived <- struct{}{}:
		default:
			// Channel is full, drop the pong signal (connection is already known to be alive)
		}
		return nil
	})
}

func (c *DefaultWebSocketClient) stopPingPong() {
	if c.pingTicker != nil {
		c.pingTicker.Stop()
	}
}

func convertToWebSocketMessageType(mt int) WebSocketMessageType {
	switch mt {
	case websocket.TextMessage:
		return WebSocketMessageTypeText
	case websocket.BinaryMessage:
		return WebSocketMessageTypeBinary
	case websocket.PingMessage:
		return WebSocketMessageTypePing
	case websocket.PongMessage:
		return WebSocketMessageTypePong
	case websocket.CloseMessage:
		return WebSocketMessageTypeClose
	default:
		return WebSocketMessageTypeBinary
	}
}

func generateSessionID() string {
	return fmt.Sprintf("ws-session-%d", time.Now().UnixNano())
}
