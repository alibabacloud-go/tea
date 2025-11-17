package dara

import (
	"context"
	"testing"
	"time"
)

// MockWebSocketHandler for testing
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

// TestWebSocketConfig tests WebSocket configuration
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

// TestWebSocketClientCreation tests client creation
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

	// Test with nil config
	_, err = NewDefaultWebSocketClient(nil, handler)
	if err == nil {
		t.Error("Expected error when config is nil")
	}

	// Test with nil handler
	_, err = NewDefaultWebSocketClient(config, nil)
	if err == nil {
		t.Error("Expected error when handler is nil")
	}
}

// TestWebSocketMessageTypes tests message type constants
func TestWebSocketMessageTypes(t *testing.T) {
	if WebSocketMessageTypeText != 0 {
		t.Errorf("Expected WebSocketMessageTypeText to be 0, got %d", WebSocketMessageTypeText)
	}

	if WebSocketMessageTypeBinary != 1 {
		t.Errorf("Expected WebSocketMessageTypeBinary to be 1, got %d", WebSocketMessageTypeBinary)
	}
}

// TestWebSocketSessionInfo tests session information
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

// TestMockHandler tests the mock handler
func TestMockHandler(t *testing.T) {
	handler := &MockWebSocketHandler{}

	session := &WebSocketSessionInfo{
		SessionID:   "test",
		ConnectedAt: time.Now(),
		Attributes:  make(map[string]interface{}),
	}

	// Test connection established
	if err := handler.AfterConnectionEstablished(session); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if !handler.ConnectedCalled {
		t.Error("AfterConnectionEstablished was not called")
	}

	// Test message handling
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

	// Test error handling
	handler.HandleError(session, context.DeadlineExceeded)
	if handler.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", handler.ErrorCount)
	}

	// Test connection closed
	handler.AfterConnectionClosed(session, 1000, "Normal")
	if !handler.ClosedCalled {
		t.Error("AfterConnectionClosed was not called")
	}

	// Test partial messages support
	if handler.SupportsPartialMessages() {
		t.Error("Expected SupportsPartialMessages to return false")
	}
}

// TestGenerateSessionID tests session ID generation
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
		{1, WebSocketMessageTypeText},    // websocket.TextMessage
		{2, WebSocketMessageTypeBinary},  // websocket.BinaryMessage
		{9, WebSocketMessageTypePing},    // websocket.PingMessage
		{10, WebSocketMessageTypePong},   // websocket.PongMessage
		{8, WebSocketMessageTypeClose},   // websocket.CloseMessage
		{999, WebSocketMessageTypeBinary}, // unknown -> binary
	}

	for _, test := range tests {
		result := convertToWebSocketMessageType(test.input)
		if result != test.expected {
			t.Errorf("convertToWebSocketMessageType(%d) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

