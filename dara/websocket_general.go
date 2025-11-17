package dara

import (
	"encoding/json"
	"fmt"
)

// GeneralMessage represents a General protocol message
type GeneralMessage struct {
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
}

// GeneralIncomingMessage represents an incoming General protocol message
type GeneralIncomingMessage struct {
	Headers    map[string]string
	Body       interface{}
	RawPayload []byte
	IsBinary   bool
}

// GeneralWebSocketHandler handles General protocol messages
type GeneralWebSocketHandler interface {
	WebSocketHandler

	// HandleGeneralTextMessage handles General protocol text messages
	HandleGeneralTextMessage(session *WebSocketSessionInfo, message *GeneralMessage) error

	// HandleGeneralBinaryMessage handles General protocol binary messages
	HandleGeneralBinaryMessage(session *WebSocketSessionInfo, data []byte) error

	// HandleGeneralIncomingMessage handles incoming General messages
	HandleGeneralIncomingMessage(session *WebSocketSessionInfo, message *GeneralIncomingMessage) error
}

// AbstractGeneralWebSocketHandler provides base implementation for General handler
type AbstractGeneralWebSocketHandler struct {
	supportsPartial bool
}

func (h *AbstractGeneralWebSocketHandler) AfterConnectionEstablished(session *WebSocketSessionInfo) error {
	return nil
}

func (h *AbstractGeneralWebSocketHandler) HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error {
	if message.Type == WebSocketMessageTypeText {
		// Parse as General text message
		generalMsg, err := ParseGeneralMessage(message)
		if err != nil {
			return err
		}

		// Create incoming message
		incoming := &GeneralIncomingMessage{
			Headers:    generalMsg.Headers,
			Body:       generalMsg.Body,
			RawPayload: message.Payload,
			IsBinary:   false,
		}

		// Call both handlers
		if err := h.HandleGeneralTextMessage(session, generalMsg); err != nil {
			return err
		}
		return h.HandleGeneralIncomingMessage(session, incoming)

	} else if message.Type == WebSocketMessageTypeBinary {
		// Handle as binary message
		incoming := &GeneralIncomingMessage{
			Headers:    make(map[string]string),
			Body:       nil,
			RawPayload: message.Payload,
			IsBinary:   true,
		}

		// Call both handlers
		if err := h.HandleGeneralBinaryMessage(session, message.Payload); err != nil {
			return err
		}
		return h.HandleGeneralIncomingMessage(session, incoming)
	}

	return nil
}

func (h *AbstractGeneralWebSocketHandler) HandleGeneralTextMessage(session *WebSocketSessionInfo, message *GeneralMessage) error {
	// Default implementation - can be overridden
	return nil
}

func (h *AbstractGeneralWebSocketHandler) HandleGeneralBinaryMessage(session *WebSocketSessionInfo, data []byte) error {
	// Default implementation - can be overridden
	return nil
}

func (h *AbstractGeneralWebSocketHandler) HandleGeneralIncomingMessage(session *WebSocketSessionInfo, message *GeneralIncomingMessage) error {
	// Default implementation - can be overridden
	return nil
}

func (h *AbstractGeneralWebSocketHandler) HandleError(session *WebSocketSessionInfo, err error) error {
	return nil
}

func (h *AbstractGeneralWebSocketHandler) AfterConnectionClosed(session *WebSocketSessionInfo, code int, reason string) error {
	return nil
}

func (h *AbstractGeneralWebSocketHandler) SupportsPartialMessages() bool {
	return h.supportsPartial
}

func (h *AbstractGeneralWebSocketHandler) SetSupportsPartialMessages(supports bool) {
	h.supportsPartial = supports
}

func ParseGeneralMessage(message *WebSocketMessage) (*GeneralMessage, error) {
	if message.Type != WebSocketMessageTypeText {
		return nil, fmt.Errorf("General text messages must be text format")
	}

	var generalMsg GeneralMessage
	if err := json.Unmarshal(message.Payload, &generalMsg); err != nil {
		// If not JSON, treat the entire payload as body
		return &GeneralMessage{
			Headers: make(map[string]string),
			Body:    string(message.Payload),
		}, nil
	}

	if generalMsg.Headers == nil {
		generalMsg.Headers = make(map[string]string)
	}

	return &generalMsg, nil
}

func BuildGeneralMessage(headers map[string]string, body interface{}) *GeneralMessage {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &GeneralMessage{
		Headers: headers,
		Body:    body,
	}
}

func BuildGeneralTextMessage(body string) *GeneralMessage {
	return BuildGeneralMessage(map[string]string{
		"Content-Type": "text/plain",
	}, body)
}

func BuildGeneralJSONMessage(body interface{}) *GeneralMessage {
	return BuildGeneralMessage(map[string]string{
		"Content-Type": "application/json",
	}, body)
}

func (m *GeneralMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *GeneralMessage) WithHeader(key, value string) *GeneralMessage {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
	return m
}

func (m *GeneralMessage) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}
