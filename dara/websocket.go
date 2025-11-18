package dara

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
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

type WebSocketConfig struct {
	URL               string
	Headers           map[string]string
	ConnectTimeout    time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	HandshakeTimeout  time.Duration
	PingInterval      time.Duration
	PongTimeout       time.Duration
	MaxMessageSize    int64
	EnableReconnect   bool
	ReconnectInterval time.Duration
	MaxReconnectTimes int
}

// WebSocketHandler handles WebSocket events and messages
type WebSocketHandler interface {
	AfterConnectionEstablished(session *WebSocketSessionInfo) error
	HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error
	HandleError(session *WebSocketSessionInfo, err error) error
	AfterConnectionClosed(session *WebSocketSessionInfo, code int, reason string) error
	SupportsPartialMessages() bool
}

type WebSocketClient interface {
	Connect(ctx context.Context) (map[string]interface{}, error)
	Disconnect(ctx context.Context) error
	Reconnect(ctx context.Context) (map[string]interface{}, error)
	IsConnected() bool
	SendText(ctx context.Context, text string) error
	SendBinary(ctx context.Context, data []byte) error
	GetSessionInfo() *WebSocketSessionInfo
	Close() error
}

type DefaultWebSocketClient struct {
	config         *WebSocketConfig
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
}

func NewDefaultWebSocketClient(config *WebSocketConfig, handler WebSocketHandler) (*DefaultWebSocketClient, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}

	client := &DefaultWebSocketClient{
		config:       config,
		handler:      handler,
		stopChan:     make(chan struct{}),
		pongReceived: make(chan struct{}, 1),
		state:        0, // disconnected
	}

	return client, nil
}

func (c *DefaultWebSocketClient) Connect(ctx context.Context) (map[string]interface{}, error) {
	atomic.StoreInt32(&c.state, 1) // connecting

	// Create a cancellable context for this connection
	// This allows us to cancel reconnect operations when the client is closed
	c.closeMu.Lock()
	if c.cancel != nil {
		c.cancel() // Cancel previous context if exists
	}
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.closeMu.Unlock()

	u, err := url.Parse(c.config.URL)
	if err != nil {
		atomic.StoreInt32(&c.state, 0) // disconnected
		return map[string]interface{}{"success": false, "error": err.Error()}, err
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.HandshakeTimeout,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}

	if u.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: false,
		}
	}

	header := http.Header{}
	for k, v := range c.config.Headers {
		header.Set(k, v)
	}

	fmt.Printf("[WebSocket] Handshake headers:\n")
	for k, v := range header {
		fmt.Printf("  %s: %v\n", k, v)
	}

	connectCtx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()

	conn, resp, err := dialer.DialContext(connectCtx, c.config.URL, header)
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

	if c.config.PingInterval > 0 {
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

	if !c.config.EnableReconnect {
		return nil, errors.New("reconnect is disabled")
	}

	if c.reconnectCount >= c.config.MaxReconnectTimes {
		return nil, fmt.Errorf("max reconnect times reached: %d", c.config.MaxReconnectTimes)
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
	case <-time.After(c.config.ReconnectInterval):
		// Continue with reconnect
	}

	// Check context again before attempting connection
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result, err := c.Connect(ctx)
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

	if c.config.WriteTimeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
	}

	return c.conn.WriteMessage(websocket.TextMessage, []byte(text))
}

func (c *DefaultWebSocketClient) SendBinary(ctx context.Context, data []byte) error {
	if !c.IsConnected() {
		return errors.New("not connected")
	}

	if c.config.WriteTimeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
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
		if c.config.ReadTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
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
				if c.config.EnableReconnect {
					// Use the connection's context so reconnect can be cancelled
					c.closeMu.Lock()
					reconnectCtx := c.ctx
					c.closeMu.Unlock()
					if reconnectCtx != nil {
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
	c.pingTicker = time.NewTicker(c.config.PingInterval)

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

				deadline := time.Now().Add(c.config.WriteTimeout)
				if err := c.conn.WriteControl(websocket.PingMessage, []byte{}, deadline); err != nil {
					if c.session != nil {
						c.handler.HandleError(c.session, err)
					}
					return
				}

				// Wait for pong
				select {
				case <-c.pongReceived:
					// Pong received
				case <-time.After(c.config.PongTimeout):
					// Pong timeout, try to reconnect
					if c.config.EnableReconnect {
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
