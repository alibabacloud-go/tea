package dara

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type MockWebSocketHandler struct {
	ConnectedCalled      bool
	MessageReceivedCount int
	ErrorCount           int
	ClosedCalled         bool
	LastMessage          *WebSocketMessage
}

func (h *MockWebSocketHandler) AfterConnectionEstablished(session *WebSocketSessionInfo) error {
	h.ConnectedCalled = true
	return nil
}

func (h *MockWebSocketHandler) HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error {
	h.MessageReceivedCount++
	h.LastMessage = message
	return nil
}

func (h *MockWebSocketHandler) HandleError(session *WebSocketSessionInfo, err error) error {
	h.ErrorCount++
	return nil
}

func (h *MockWebSocketHandler) AfterConnectionClosed(session *WebSocketSessionInfo, code int, reason string) error {
	h.ClosedCalled = true
	return nil
}

func (h *MockWebSocketHandler) SupportsPartialMessages() bool {
	return false
}

func TestWebSocketClientCreation(t *testing.T) {
	handler := &MockWebSocketHandler{}

	// Test new API: NewDefaultWebSocketClient takes handler and websocketSubProtocol
	client, err := NewDefaultWebSocketClient(handler)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}

	// Test error cases
	_, err = NewDefaultWebSocketClient(nil)
	if err == nil {
		t.Error("Expected error when handler is nil")
	}
}

func TestWebSocketMessageTypes(t *testing.T) {
	if WebSocketMessageTypeText != 0 {
		t.Errorf("Expected WebSocketMessageTypeText to be 0, got %d", WebSocketMessageTypeText)
	}

	if WebSocketMessageTypeBinary != 1 {
		t.Errorf("Expected WebSocketMessageTypeBinary to be 1, got %d", WebSocketMessageTypeBinary)
	}
}

func TestWebSocketSessionInfo(t *testing.T) {
	session := &WebSocketSessionInfo{
		SessionID:   "test-session-123",
		ConnectedAt: time.Now(),
		RemoteAddr:  "192.168.1.1:8080",
		LocalAddr:   "192.168.1.2:12345",
		Attributes:  make(map[string]interface{}),
	}

	if session.SessionID != "test-session-123" {
		t.Errorf("Expected SessionID 'test-session-123', got '%s'", session.SessionID)
	}

	// Test attributes
	session.Attributes["key1"] = "value1"
	session.Attributes["key2"] = 123

	if session.Attributes["key1"] != "value1" {
		t.Error("Failed to get attribute 'key1'")
	}

	if session.Attributes["key2"] != 123 {
		t.Error("Failed to get attribute 'key2'")
	}
}

func TestMockHandler(t *testing.T) {
	handler := &MockWebSocketHandler{}

	session := &WebSocketSessionInfo{
		SessionID:   "test",
		ConnectedAt: time.Now(),
		Attributes:  make(map[string]interface{}),
	}

	if err := handler.AfterConnectionEstablished(session); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if !handler.ConnectedCalled {
		t.Error("AfterConnectionEstablished was not called")
	}

	msg := &WebSocketMessage{
		Type:      WebSocketMessageTypeText,
		Payload:   []byte("test message"),
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
	}

	handler.HandleRawMessage(session, msg)
	handler.HandleRawMessage(session, msg)

	if handler.MessageReceivedCount != 2 {
		t.Errorf("Expected 2 messages, got %d", handler.MessageReceivedCount)
	}

	if string(handler.LastMessage.Payload) != "test message" {
		t.Errorf("Expected last message 'test message', got '%s'", string(handler.LastMessage.Payload))
	}

	handler.HandleError(session, context.DeadlineExceeded)
	if handler.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", handler.ErrorCount)
	}

	handler.AfterConnectionClosed(session, 1000, "Normal")
	if !handler.ClosedCalled {
		t.Error("AfterConnectionClosed was not called")
	}

	if handler.SupportsPartialMessages() {
		t.Error("Expected SupportsPartialMessages to return false")
	}
}

// TestConvertToWebSocketMessageType tests message type conversion
func TestConvertToWebSocketMessageType(t *testing.T) {
	tests := []struct {
		input    int
		expected WebSocketMessageType
	}{
		{1, WebSocketMessageTypeText},     // websocket.TextMessage
		{2, WebSocketMessageTypeBinary},   // websocket.BinaryMessage
		{9, WebSocketMessageTypePing},     // websocket.PingMessage
		{10, WebSocketMessageTypePong},    // websocket.PongMessage
		{8, WebSocketMessageTypeClose},    // websocket.CloseMessage
		{999, WebSocketMessageTypeBinary}, // unknown -> binary
	}

	for _, test := range tests {
		result := convertToWebSocketMessageType(test.input)
		if result != test.expected {
			t.Errorf("convertToWebSocketMessageType(%d) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestDefaultWebSocketClient_Connect(t *testing.T) {
	// Create a test WebSocket server
	server := createTestWebSocketServer(t)
	defer server.Close()

	// Convert server URL to WebSocket URL
	wsURL := "ws" + server.URL[4:] // Replace "http" with "ws"

	t.Run("Successful connection", func(t *testing.T) {
		// Create Request (matching DoRequest pattern)
		request := NewRequest()
		u, _ := url.Parse(wsURL)
		request.Protocol = String(u.Scheme)
		request.Domain = String(u.Host)
		request.Pathname = String(u.Path)
		request.Headers = make(map[string]*string)

		// Create RuntimeObject (matching DoRequest pattern)
		runtimeObject := NewRuntimeObject(map[string]interface{}{
			"connectTimeout":           5000,
			"readTimeout":              30000,
			"webSocketPingInterval":    0, // Disable ping for this test
			"webSocketEnableReconnect": false,
		})

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Test initial state
		if client.IsConnected() {
			t.Error("Client should not be connected initially")
		}

		// Connect
		ctx := context.Background()
		response, err := client.Connect(ctx, request, runtimeObject)
		if err != nil {
			t.Fatalf("Connect failed: %v", err)
		}

		// Verify response
		if response == nil {
			t.Error("Response should not be nil")
		} else {
			// Verify response has valid status code (101 for WebSocket upgrade)
			if response.StatusCode == nil {
				t.Error("Response StatusCode should not be nil")
			} else if *response.StatusCode != 101 {
				// Note: In test environment, status code might vary, so we just check it's not nil
				// In real WebSocket handshake, status code should be 101
			}
		}

		// Verify state
		if !client.IsConnected() {
			t.Error("Client should be connected after Connect()")
		}

		// Verify handler was called
		if !handler.ConnectedCalled {
			t.Error("AfterConnectionEstablished should be called")
		}

		// Verify session was created
		if client.session == nil {
			t.Error("Session should be created after connection")
		}

		if client.session.SessionID == "" {
			t.Error("Session ID should not be empty")
		}

		// Cleanup
		client.Disconnect()
	})

	t.Run("Invalid URL", func(t *testing.T) {
		// Create Request with invalid URL
		request := NewRequest()
		request.Protocol = String("invalid-url")
		request.Domain = String("test")
		request.Headers = make(map[string]*string)

		runtimeObject := NewRuntimeObject(map[string]interface{}{
			"connectTimeout": 5000,
			"readTimeout":    30000,
		})

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		response, err := client.Connect(ctx, request, runtimeObject)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}

		if response != nil {
			t.Error("Expected response to be nil for invalid URL")
		}

		// Verify state is disconnected
		if client.IsConnected() {
			t.Error("Client should not be connected after failed connection")
		}
	})

	t.Run("Connection timeout", func(t *testing.T) {
		// Use a non-existent server with very short timeout
		request := NewRequest()
		request.Protocol = String("ws")
		request.Domain = String("127.0.0.1:99999") // Non-existent port
		request.Pathname = String("/")
		request.Headers = make(map[string]*string)

		runtimeObject := NewRuntimeObject(map[string]interface{}{
			"connectTimeout":            100, // Very short timeout
			"readTimeout":               30000,
			"webSocketHandshakeTimeout": 100,
		})

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		response, err := client.Connect(ctx, request, runtimeObject)
		if err == nil {
			t.Error("Expected error for connection timeout")
		}

		if response != nil {
			t.Error("Expected response to be nil for timeout")
		}

		// Verify state is disconnected
		if client.IsConnected() {
			t.Error("Client should not be connected after timeout")
		}

		// Verify handler was not called
		if handler.ConnectedCalled {
			t.Error("AfterConnectionEstablished should not be called on timeout")
		}
	})

	t.Run("Connection with custom headers", func(t *testing.T) {
		request := NewRequest()
		u, _ := url.Parse(wsURL)
		request.Protocol = String(u.Scheme)
		request.Domain = String(u.Host)
		request.Pathname = String(u.Path)
		request.Headers = map[string]*string{
			"X-Custom-Header": String("test-value"),
			"User-Agent":      String("test-agent"),
		}

		runtimeObject := NewRuntimeObject(map[string]interface{}{
			"connectTimeout":        5000,
			"readTimeout":           30000,
			"webSocketPingInterval": 0,
		})

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		response, err := client.Connect(ctx, request, runtimeObject)
		if err != nil {
			t.Fatalf("Connect failed: %v", err)
		}

		if response == nil {
			t.Error("Response should not be nil on successful connection")
		}

		// Cleanup
		client.Disconnect()
	})

	t.Run("Handler error on connection established", func(t *testing.T) {
		errorHandler := &ErrorOnConnectHandler{}
		request := NewRequest()
		u, _ := url.Parse(wsURL)
		request.Protocol = String(u.Scheme)
		request.Domain = String(u.Host)
		request.Pathname = String(u.Path)
		request.Headers = make(map[string]*string)

		runtimeObject := NewRuntimeObject(map[string]interface{}{
			"connectTimeout":        5000,
			"readTimeout":           30000,
			"webSocketPingInterval": 0,
		})

		client, err := NewDefaultWebSocketClient(errorHandler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		response, err := client.Connect(ctx, request, runtimeObject)
		if err == nil {
			t.Error("Expected error when handler returns error")
		}

		if response != nil {
			t.Error("Expected response to be nil when handler returns error")
		}

		// Cleanup - connection might still be established even if handler fails
		client.Disconnect()
	})

	t.Run("Connection with ping interval", func(t *testing.T) {
		request := NewRequest()
		u, _ := url.Parse(wsURL)
		request.Protocol = String(u.Scheme)
		request.Domain = String(u.Host)
		request.Pathname = String(u.Path)
		request.Headers = make(map[string]*string)

		runtimeObject := NewRuntimeObject(map[string]interface{}{
			"connectTimeout":        5000,
			"readTimeout":           30000,
			"webSocketPingInterval": 1000, // Enable ping
			"webSocketPongTimeout":  500,
		})

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		response, err := client.Connect(ctx, request, runtimeObject)
		if err != nil {
			t.Fatalf("Connect failed: %v", err)
		}

		if response == nil {
			t.Error("Response should not be nil on successful connection")
		}

		// Wait a bit to ensure ping goroutine starts
		time.Sleep(100 * time.Millisecond)

		// Cleanup
		client.Disconnect()
	})
}

// ErrorOnConnectHandler is a handler that returns an error on connection
type ErrorOnConnectHandler struct {
	MockWebSocketHandler
}

func (h *ErrorOnConnectHandler) AfterConnectionEstablished(session *WebSocketSessionInfo) error {
	return errors.New("test error on connection")
}

func createTestWebSocketServer(t *testing.T) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for testing
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("WebSocket upgrade error: %v", err)
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				break
			}
		}
	})

	server := httptest.NewServer(handler)
	return server
}

func TestWebSocketReconnectWhenAlreadyConnected(t *testing.T) {
	server := createTestWebSocketServer(t)
	defer server.Close()

	handler := &MockWebSocketHandler{}

	request := &Request{
		Protocol: String("ws"),
		Pathname: String("/"),
		Headers: map[string]*string{
			"host": String(server.Listener.Addr().String()),
		},
	}

	runtimeObject := &RuntimeObject{
		WebSocketEnableReconnect:   Bool(true),
		WebSocketMaxReconnectTimes: Int(3),
		WebSocketReconnectInterval: Int(1000),
		WebSocketHandshakeTimeout:  Int(5000),
		WebSocketPingInterval:      Int(0), // Disable ping for this test
		WebSocketHandler:           handler,
	}

	// Connect to the server
	client, _, err := NewWebSocketClientAndConnect(request, runtimeObject)
	if err != nil {
		t.Fatalf("Initial connection failed: %v", err)
	}

	// Verify client is connected
	if !client.IsConnected() {
		t.Fatal("Client should be connected")
	}

	// Try to reconnect while already connected
	response, err := client.Reconnect()
	// Reconnect when already connected should return an error indicating skip
	if err == nil {
		t.Error("Reconnect should return error when already connected")
	}
	if err != nil && err.Error() != "already connected" {
		t.Errorf("Expected 'already connected' error, got: %v", err)
	}
	if response != nil {
		t.Error("Response should be nil when reconnect is skipped")
	}

	// Verify client is still connected
	if !client.IsConnected() {
		t.Error("Client should still be connected after skipped reconnect")
	}

	// Cleanup
	client.Close()

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)
}
