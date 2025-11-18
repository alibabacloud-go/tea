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
	IsConnected() bool
	SendText(ctx context.Context, text string) error
	SendBinary(ctx context.Context, data []byte) error
	GetSessionInfo() *WebSocketSessionInfo
	Close() error
}

type DefaultWebSocketClient struct {
	handler        WebSocketHandler
	conn           *websocket.Conn
	session        *WebSocketSessionInfo
	state          int32 // 0=disconnected, 1=connecting, 2=connected, 3=disconnecting
	reconnectCount int
	reconnectMu    sync.Mutex
	stopChan       chan struct{}
	pingTicker     *time.Ticker
	pongReceived   chan struct{}
	wg             sync.WaitGroup
	closeMu        sync.Mutex
	closed         bool
	ctx            context.Context    // Context for the connection lifecycle
	cancel         context.CancelFunc // Cancel function for the context
	request        *Request
	runtimeObject  *RuntimeObject
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

func buildWebSocketURL(request *Request) (string, error) {
	if request == nil {
		return "", errors.New("request cannot be nil")
	}

	// Set default protocol
	if request.Protocol == nil {
		request.Protocol = String("ws")
	} else {
		protocol := strings.ToLower(StringValue(request.Protocol))
		// Convert http/https to ws/wss
		if protocol == "http" {
			protocol = "ws"
		} else if protocol == "https" {
			protocol = "wss"
		}
		request.Protocol = String(protocol)
	}

	// Get domain from headers["host"] or Domain field
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

func (c *DefaultWebSocketClient) Connect(ctx context.Context, request *Request, runtimeObject *RuntimeObject) (map[string]interface{}, error) {
	if request == nil {
		return map[string]interface{}{"success": false, "error": "request cannot be nil"}, errors.New("request cannot be nil")
	}
	if runtimeObject == nil {
		runtimeObject = &RuntimeObject{}
	}

	// Store request and runtimeObject for reconnection
	c.request = request
	c.runtimeObject = runtimeObject

	atomic.StoreInt32(&c.state, 1) // connecting

	// Create a cancellable context for this connection
	// This allows us to cancel reconnect operations when the client is closed
	c.closeMu.Lock()
	if c.cancel != nil {
		c.cancel() // Cancel previous context if exists
	}
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.closeMu.Unlock()

	// Build WebSocket URL from Request (matching DoRequest pattern)
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

	// Get WebSocket-specific configuration from RuntimeObject
	handshakeTimeout := time.Duration(IntValue(runtimeObject.WebSocketHandshakeTimeout)) * time.Millisecond
	if handshakeTimeout <= 0 {
		handshakeTimeout = 30 * time.Second // Default
	}
	connectTimeout := time.Duration(IntValue(runtimeObject.ConnectTimeout)) * time.Millisecond
	if connectTimeout <= 0 {
		connectTimeout = 10 * time.Second // Default
	}

	// Configure dialer with proxy, SSL/TLS, and network settings (matching DoRequest)
	dialer := websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}

	// Configure TLS/SSL (matching DoRequest's getHttpTransport)
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

		// Check NoProxy list (matching DoRequest's getNoProxy)
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

		// Configure LocalAddr (matching DoRequest's setDialContext)
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

	// Build headers from Request (matching DoRequest pattern)
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

	// This avoids race condition where connection might be closed before handler is set
	c.setupPongHandler()

	// HTTP headers are case-insensitive, but Go's Header.Get() is case-insensitive
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
	pingInterval := time.Duration(IntValue(runtimeObject.WebSocketPingInterval)) * time.Millisecond
	if pingInterval > 0 {
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
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	// Check if context is already cancelled (client might be closing)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if c.runtimeObject == nil || !BoolValue(c.runtimeObject.WebSocketEnableReconnect) {
		return nil, errors.New("reconnect is disabled")
	}

	maxReconnectTimes := IntValue(c.runtimeObject.WebSocketMaxReconnectTimes)
	if maxReconnectTimes <= 0 {
		maxReconnectTimes = 5 // Default
	}
	if c.reconnectCount >= maxReconnectTimes {
		return nil, fmt.Errorf("max reconnect times reached: %d", maxReconnectTimes)
	}

	// Clean up resources before reconnecting
	c.cleanupResources()

	c.closed = false
	c.stopChan = make(chan struct{})
	c.reconnectCount++

	// Use context-aware sleep to allow cancellation during reconnect interval
	reconnectInterval := time.Duration(IntValue(c.runtimeObject.WebSocketReconnectInterval)) * time.Millisecond
	if reconnectInterval <= 0 {
		reconnectInterval = 5 * time.Second // Default
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(reconnectInterval):
		// Continue with reconnect
	}

	// Check context again before attempting connection
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if c.request == nil || c.runtimeObject == nil {
		return nil, errors.New("request or runtimeObject is nil, cannot reconnect")
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

	if c.runtimeObject != nil {
		writeTimeout := time.Duration(IntValue(c.runtimeObject.WebSocketWriteTimeout)) * time.Millisecond
		if writeTimeout > 0 {
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		}
	}

	return c.conn.WriteMessage(websocket.TextMessage, []byte(text))
}

func (c *DefaultWebSocketClient) SendBinary(ctx context.Context, data []byte) error {
	if !c.IsConnected() {
		return errors.New("not connected")
	}

	if c.runtimeObject != nil {
		writeTimeout := time.Duration(IntValue(c.runtimeObject.WebSocketWriteTimeout)) * time.Millisecond
		if writeTimeout > 0 {
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		}
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
		if c.runtimeObject != nil {
			readTimeout := time.Duration(IntValue(c.runtimeObject.ReadTimeout)) * time.Millisecond
			if readTimeout > 0 {
				c.conn.SetReadDeadline(time.Now().Add(readTimeout))
			}
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
				if c.runtimeObject != nil && BoolValue(c.runtimeObject.WebSocketEnableReconnect) {
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
			// Check if handler is an AWAP handler - if so, it will handle the message through HandleRawMessage
			// which will parse and route to HandleAwapMessage/HandleAwapIncomingMessage
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
	if c.runtimeObject == nil {
		return
	}
	pingInterval := time.Duration(IntValue(c.runtimeObject.WebSocketPingInterval)) * time.Millisecond
	if pingInterval <= 0 {
		return
	}
	c.pingTicker = time.NewTicker(pingInterval)

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

				writeTimeout := time.Duration(IntValue(c.runtimeObject.WebSocketWriteTimeout)) * time.Millisecond
				if writeTimeout <= 0 {
					writeTimeout = 30 * time.Second // Default
				}
				deadline := time.Now().Add(writeTimeout)
				if err := c.conn.WriteControl(websocket.PingMessage, []byte{}, deadline); err != nil {
					if c.session != nil {
						c.handler.HandleError(c.session, err)
					}
					return
				}

				pongTimeout := time.Duration(IntValue(c.runtimeObject.WebSocketPongTimeout)) * time.Millisecond
				if pongTimeout <= 0 {
					pongTimeout = 10 * time.Second // Default
				}
				select {
				case <-c.pongReceived:
					// Pong received
				case <-time.After(pongTimeout):
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

// setupPongHandler sets up the pong handler for the WebSocket connection
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
