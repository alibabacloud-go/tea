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

// WebSocketMessageType represents the type of WebSocket message
type WebSocketMessageType int

const (
	WebSocketMessageTypeText WebSocketMessageType = iota
	WebSocketMessageTypeBinary
	WebSocketMessageTypePing
	WebSocketMessageTypePong
	WebSocketMessageTypeClose
)

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      WebSocketMessageType
	Payload   []byte
	Headers   map[string]string
	Timestamp time.Time
}

// WebSocketCloseFrame represents a close frame
type WebSocketCloseFrame struct {
	Code   int
	Reason string
}

// WebSocketSessionInfo holds information about a WebSocket session
type WebSocketSessionInfo struct {
	SessionID   string
	ConnectedAt time.Time
	RemoteAddr  string
	LocalAddr   string
	Attributes  map[string]interface{}
}

// WebSocketConfig holds configuration for WebSocket client
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

// WebSocketClient is the WebSocket client interface
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

// DefaultWebSocketClient is the default implementation of WebSocketClient
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
}

// NewDefaultWebSocketClient creates a new DefaultWebSocketClient
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

// Connect establishes a WebSocket connection
func (c *DefaultWebSocketClient) Connect(ctx context.Context) (map[string]interface{}, error) {
	atomic.StoreInt32(&c.state, 1) // connecting

	u, err := url.Parse(c.config.URL)
	if err != nil {
		atomic.StoreInt32(&c.state, 0) // disconnected
		return map[string]interface{}{"success": false, "error": err.Error()}, err
	}

	// Setup dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.HandshakeTimeout,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}

	// Add TLS config if needed
	if u.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: false,
		}
	}

	// Setup headers
	header := http.Header{}
	for k, v := range c.config.Headers {
		header.Set(k, v)
	}

	// Connect with timeout
	connectCtx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()

	conn, resp, err := dialer.DialContext(connectCtx, c.config.URL, header)
	if err != nil {
		atomic.StoreInt32(&c.state, 0) // disconnected
		return map[string]interface{}{"success": false, "error": err.Error()}, err
	}

	c.conn = conn
	atomic.StoreInt32(&c.state, 2) // connected

	// Create session
	c.session = &WebSocketSessionInfo{
		SessionID:   generateSessionID(),
		ConnectedAt: time.Now(),
		RemoteAddr:  conn.RemoteAddr().String(),
		LocalAddr:   conn.LocalAddr().String(),
		Attributes:  make(map[string]interface{}),
	}

	// Start message handlers
	c.startMessageHandlers()

	// Start ping/pong
	if c.config.PingInterval > 0 {
		c.startPingPong()
	}

	// Call handler
	if err := c.handler.AfterConnectionEstablished(c.session); err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}, err
	}

	result := map[string]interface{}{
		"success": true,
		"status":  resp.StatusCode,
		"header":  resp.Header,
	}

	return result, nil
}

// Disconnect closes the WebSocket connection
func (c *DefaultWebSocketClient) Disconnect(ctx context.Context) error {
	return c.disconnect(1000, "Normal closure")
}

// disconnect closes the connection with a specific code and reason
func (c *DefaultWebSocketClient) disconnect(code int, reason string) error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}

	atomic.StoreInt32(&c.state, 3) // disconnecting

	// Stop ping/pong
	c.stopPingPong()

	// Close connection
	if c.conn != nil {
		deadline := time.Now().Add(time.Second)
		c.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(code, reason),
			deadline)
		c.conn.Close()
	}

	// Call handler
	if c.session != nil {
		c.handler.AfterConnectionClosed(c.session, code, reason)
	}

	atomic.StoreInt32(&c.state, 0) // disconnected
	c.closed = true

	// Stop message handlers
	close(c.stopChan)
	c.wg.Wait()

	return nil
}

// Reconnect reconnects the WebSocket
func (c *DefaultWebSocketClient) Reconnect(ctx context.Context) (map[string]interface{}, error) {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	if !c.config.EnableReconnect {
		return nil, errors.New("reconnect is disabled")
	}

	if c.reconnectCount >= c.config.MaxReconnectTimes {
		return nil, fmt.Errorf("max reconnect times reached: %d", c.config.MaxReconnectTimes)
	}

	// Close existing connection
	if c.conn != nil {
		c.conn.Close()
	}

	// Reset state
	c.closed = false
	c.stopChan = make(chan struct{})
	c.reconnectCount++

	// Wait before reconnecting
	time.Sleep(c.config.ReconnectInterval)

	// Try to connect
	result, err := c.Connect(ctx)
	if err == nil {
		c.reconnectCount = 0 // Reset on success
	}

	return result, err
}

// IsConnected returns whether the client is connected
func (c *DefaultWebSocketClient) IsConnected() bool {
	return atomic.LoadInt32(&c.state) == 2
}

// SendText sends a text message
func (c *DefaultWebSocketClient) SendText(ctx context.Context, text string) error {
	if !c.IsConnected() {
		return errors.New("not connected")
	}

	if c.config.WriteTimeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
	}

	return c.conn.WriteMessage(websocket.TextMessage, []byte(text))
}

// SendBinary sends a binary message
func (c *DefaultWebSocketClient) SendBinary(ctx context.Context, data []byte) error {
	if !c.IsConnected() {
		return errors.New("not connected")
	}

	if c.config.WriteTimeout > 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
	}

	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}

// GetSessionInfo returns the current session information
func (c *DefaultWebSocketClient) GetSessionInfo() *WebSocketSessionInfo {
	return c.session
}

// Close closes the client and releases resources
func (c *DefaultWebSocketClient) Close() error {
	return c.disconnect(1000, "Client closed")
}

// startMessageHandlers starts goroutines to handle incoming messages
func (c *DefaultWebSocketClient) startMessageHandlers() {
	// Read messages from WebSocket
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.readMessages()
	}()
}

// readMessages reads messages from WebSocket connection
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
		select {
		case <-c.stopChan:
			return
		default:
			if c.conn == nil {
				return
			}

			// Set read deadline
			if c.config.ReadTimeout > 0 {
				c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
			}

			messageType, data, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					if c.session != nil {
						c.handler.HandleError(c.session, err)
					}
				}

				// Try to reconnect
				if c.config.EnableReconnect {
					go c.Reconnect(context.Background())
				}
				return
			}

			// Convert to WebSocketMessage
			msg := &WebSocketMessage{
				Type:      convertToWebSocketMessageType(messageType),
				Payload:   data,
				Headers:   make(map[string]string),
				Timestamp: time.Now(),
			}

			// Handle message
			if c.session != nil {
				if err := c.handler.HandleRawMessage(c.session, msg); err != nil {
					c.handler.HandleError(c.session, err)
				}
			}
		}
	}
}

// startPingPong starts ping/pong mechanism
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

				// Send ping
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
						go c.Reconnect(context.Background())
					}
					return
				}
			}
		}
	}()

	// Setup pong handler
	if c.conn != nil {
		c.conn.SetPongHandler(func(appData string) error {
			select {
			case c.pongReceived <- struct{}{}:
			default:
			}
			return nil
		})
	}
}

// stopPingPong stops ping/pong mechanism
func (c *DefaultWebSocketClient) stopPingPong() {
	if c.pingTicker != nil {
		c.pingTicker.Stop()
	}
}

// convertToWebSocketMessageType converts gorilla/websocket message type
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

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("ws-session-%d", time.Now().UnixNano())
}

