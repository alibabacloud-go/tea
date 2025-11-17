package dara

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestWebSocketConfig(t *testing.T) {
	config := &WebSocketConfig{
		URL:               "ws://localhost:8080",
		Headers:           make(map[string]string),
		ConnectTimeout:    10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      10 * time.Second,
		HandshakeTimeout:  5 * time.Second,
		PingInterval:      15 * time.Second,
		PongTimeout:       5 * time.Second,
		MaxMessageSize:    1024 * 1024,
		EnableReconnect:   true,
		ReconnectInterval: 3 * time.Second,
		MaxReconnectTimes: 5,
	}

	if config.URL != "ws://localhost:8080" {
		t.Errorf("Expected URL 'ws://localhost:8080', got '%s'", config.URL)
	}

	if config.EnableReconnect != true {
		t.Error("Expected EnableReconnect to be true")
	}
}

func TestWebSocketClientCreation(t *testing.T) {
	config := &WebSocketConfig{
		URL:               "ws://localhost:8080",
		Headers:           make(map[string]string),
		ConnectTimeout:    10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      10 * time.Second,
		EnableReconnect:   true,
		ReconnectInterval: 3 * time.Second,
		MaxReconnectTimes: 5,
	}

	handler := &MockWebSocketHandler{}

	client, err := NewDefaultWebSocketClient(config, handler)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}

	_, err = NewDefaultWebSocketClient(nil, handler)
	if err == nil {
		t.Error("Expected error when config is nil")
	}

	_, err = NewDefaultWebSocketClient(config, nil)
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

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	if id1 == "" {
		t.Error("generateSessionID should not return empty string")
	}

	// Test that IDs are unique
	id2 := generateSessionID()
	if id1 == id2 {
		t.Error("Multiple calls to generateSessionID should return different IDs")
	}

	// Test ID format
	if len(id1) < 10 {
		t.Error("Session ID should be reasonably long")
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
		config := &WebSocketConfig{
			URL:              wsURL,
			Headers:          make(map[string]string),
			ConnectTimeout:   5 * time.Second,
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     10 * time.Second,
			HandshakeTimeout: 5 * time.Second,
			PingInterval:     0, // Disable ping for this test
			EnableReconnect:  false,
		}

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(config, handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Test initial state
		if client.IsConnected() {
			t.Error("Client should not be connected initially")
		}

		// Connect
		ctx := context.Background()
		result, err := client.Connect(ctx)
		if err != nil {
			t.Fatalf("Connect failed: %v", err)
		}

		// Verify result
		if success, ok := result["success"].(bool); !ok || !success {
			t.Errorf("Expected success=true, got %v", result)
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
		client.Disconnect(ctx)
	})

	t.Run("Invalid URL", func(t *testing.T) {
		config := &WebSocketConfig{
			URL:              "invalid-url://test",
			Headers:          make(map[string]string),
			ConnectTimeout:   5 * time.Second,
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     10 * time.Second,
			HandshakeTimeout: 5 * time.Second,
		}

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(config, handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		result, err := client.Connect(ctx)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}

		if success, ok := result["success"].(bool); ok && success {
			t.Error("Expected success=false for invalid URL")
		}

		// Verify state is disconnected
		if client.IsConnected() {
			t.Error("Client should not be connected after failed connection")
		}
	})

	t.Run("Connection timeout", func(t *testing.T) {
		// Use a non-existent server with very short timeout
		config := &WebSocketConfig{
			URL:              "ws://127.0.0.1:99999", // Non-existent port
			Headers:          make(map[string]string),
			ConnectTimeout:   100 * time.Millisecond, // Very short timeout
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     10 * time.Second,
			HandshakeTimeout: 100 * time.Millisecond,
		}

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(config, handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		result, err := client.Connect(ctx)
		if err == nil {
			t.Error("Expected error for connection timeout")
		}

		if success, ok := result["success"].(bool); ok && success {
			t.Error("Expected success=false for timeout")
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
		config := &WebSocketConfig{
			URL: wsURL,
			Headers: map[string]string{
				"X-Custom-Header": "test-value",
				"User-Agent":      "test-agent",
			},
			ConnectTimeout:   5 * time.Second,
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     10 * time.Second,
			HandshakeTimeout: 5 * time.Second,
			PingInterval:     0,
		}

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(config, handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		result, err := client.Connect(ctx)
		if err != nil {
			t.Fatalf("Connect failed: %v", err)
		}

		if success, ok := result["success"].(bool); !ok || !success {
			t.Errorf("Expected success=true, got %v", result)
		}

		// Cleanup
		client.Disconnect(ctx)
	})

	t.Run("Handler error on connection established", func(t *testing.T) {
		errorHandler := &ErrorOnConnectHandler{}
		config := &WebSocketConfig{
			URL:              wsURL,
			Headers:          make(map[string]string),
			ConnectTimeout:   5 * time.Second,
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     10 * time.Second,
			HandshakeTimeout: 5 * time.Second,
			PingInterval:     0,
		}

		client, err := NewDefaultWebSocketClient(config, errorHandler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		result, err := client.Connect(ctx)
		if err == nil {
			t.Error("Expected error when handler returns error")
		}

		if success, ok := result["success"].(bool); ok && success {
			t.Error("Expected success=false when handler returns error")
		}

		// Cleanup - connection might still be established even if handler fails
		client.Disconnect(ctx)
	})

	t.Run("Connection with ping interval", func(t *testing.T) {
		config := &WebSocketConfig{
			URL:              wsURL,
			Headers:          make(map[string]string),
			ConnectTimeout:   5 * time.Second,
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     10 * time.Second,
			HandshakeTimeout: 5 * time.Second,
			PingInterval:     1 * time.Second, // Enable ping
			PongTimeout:      500 * time.Millisecond,
		}

		handler := &MockWebSocketHandler{}
		client, err := NewDefaultWebSocketClient(config, handler)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		result, err := client.Connect(ctx)
		if err != nil {
			t.Fatalf("Connect failed: %v", err)
		}

		if success, ok := result["success"].(bool); !ok || !success {
			t.Errorf("Expected success=true, got %v", result)
		}

		// Wait a bit to ensure ping goroutine starts
		time.Sleep(100 * time.Millisecond)

		// Cleanup
		client.Disconnect(ctx)
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
