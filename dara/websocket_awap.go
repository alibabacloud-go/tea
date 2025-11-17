package dara

import (
	"encoding/json"
	"fmt"
)

// AwapMessageType represents AWAP message types
type AwapMessageType string

const (
	AwapMessageTypeRequest  AwapMessageType = "request"
	AwapMessageTypeResponse AwapMessageType = "response"
	AwapMessageTypeEvent    AwapMessageType = "event"
)

// AwapMessageFormat represents AWAP message format
type AwapMessageFormat string

const (
	AwapMessageFormatText   AwapMessageFormat = "text"
	AwapMessageFormatBinary AwapMessageFormat = "binary"
)

// AwapMessage represents an AWAP protocol message
type AwapMessage struct {
	Type    AwapMessageType            `json:"type"`
	ID      string                     `json:"id"`
	Seq     int64                      `json:"seq"`
	Headers map[string]string          `json:"headers,omitempty"`
	Payload interface{}                `json:"payload,omitempty"`
	Format  AwapMessageFormat          `json:"format,omitempty"`
	Status  int                        `json:"status,omitempty"`
	Error   string                     `json:"error,omitempty"`
	Data    map[string]interface{}     `json:"data,omitempty"`
}

// AwapIncomingMessage represents an incoming AWAP message
type AwapIncomingMessage struct {
	AwapMessage
	RawPayload []byte
}

// AwapWebSocketHandler handles AWAP protocol messages
type AwapWebSocketHandler interface {
	WebSocketHandler
	
	// HandleAwapMessage handles AWAP protocol messages
	HandleAwapMessage(session *WebSocketSessionInfo, message *AwapMessage) error
	
	// HandleAwapIncomingMessage handles incoming AWAP messages
	HandleAwapIncomingMessage(session *WebSocketSessionInfo, message *AwapIncomingMessage) error
}

// AbstractAwapWebSocketHandler provides base implementation for AWAP handler
type AbstractAwapWebSocketHandler struct {
	supportsPartial bool
}

// AfterConnectionEstablished is called after connection is established
func (h *AbstractAwapWebSocketHandler) AfterConnectionEstablished(session *WebSocketSessionInfo) error {
	return nil
}

// HandleRawMessage processes raw WebSocket messages and converts to AWAP format
func (h *AbstractAwapWebSocketHandler) HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error {
	// Parse as AWAP message
	awapMsg, err := ParseAwapMessage(message)
	if err != nil {
		return err
	}
	
	// Handle based on message type
	if awapMsg.Type == AwapMessageTypeEvent {
		incoming := &AwapIncomingMessage{
			AwapMessage: *awapMsg,
			RawPayload:  message.Payload,
		}
		return h.HandleAwapIncomingMessage(session, incoming)
	}
	
	return h.HandleAwapMessage(session, awapMsg)
}

// HandleAwapMessage handles AWAP protocol messages (default implementation)
func (h *AbstractAwapWebSocketHandler) HandleAwapMessage(session *WebSocketSessionInfo, message *AwapMessage) error {
	// Default implementation - can be overridden
	return nil
}

// HandleAwapIncomingMessage handles incoming AWAP messages (default implementation)
func (h *AbstractAwapWebSocketHandler) HandleAwapIncomingMessage(session *WebSocketSessionInfo, message *AwapIncomingMessage) error {
	// Default implementation - can be overridden
	return nil
}

// HandleError handles errors
func (h *AbstractAwapWebSocketHandler) HandleError(session *WebSocketSessionInfo, err error) error {
	return nil
}

// AfterConnectionClosed is called after connection is closed
func (h *AbstractAwapWebSocketHandler) AfterConnectionClosed(session *WebSocketSessionInfo, code int, reason string) error {
	return nil
}

// SupportsPartialMessages returns whether partial messages are supported
func (h *AbstractAwapWebSocketHandler) SupportsPartialMessages() bool {
	return h.supportsPartial
}

// SetSupportsPartialMessages sets whether to support partial messages
func (h *AbstractAwapWebSocketHandler) SetSupportsPartialMessages(supports bool) {
	h.supportsPartial = supports
}

// ParseAwapMessage parses a WebSocket message as AWAP format
func ParseAwapMessage(message *WebSocketMessage) (*AwapMessage, error) {
	if message.Type != WebSocketMessageTypeText {
		return nil, fmt.Errorf("AWAP messages must be text format")
	}
	
	var awapMsg AwapMessage
	if err := json.Unmarshal(message.Payload, &awapMsg); err != nil {
		return nil, fmt.Errorf("failed to parse AWAP message: %w", err)
	}
	
	return &awapMsg, nil
}

// BuildAwapMessage builds an AWAP message
func BuildAwapMessage(msgType AwapMessageType, id string, seq int64, payload interface{}) *AwapMessage {
	return &AwapMessage{
		Type:    msgType,
		ID:      id,
		Seq:     seq,
		Payload: payload,
		Headers: make(map[string]string),
	}
}

// BuildAwapRequest builds an AWAP request message
func BuildAwapRequest(id string, seq int64, payload interface{}) *AwapMessage {
	return BuildAwapMessage(AwapMessageTypeRequest, id, seq, payload)
}

// BuildAwapResponse builds an AWAP response message
func BuildAwapResponse(id string, seq int64, status int, data interface{}) *AwapMessage {
	msg := BuildAwapMessage(AwapMessageTypeResponse, id, seq, nil)
	msg.Status = status
	msg.Data = map[string]interface{}{"result": data}
	return msg
}

// BuildAwapEvent builds an AWAP event message
func BuildAwapEvent(id string, seq int64, payload interface{}) *AwapMessage {
	return BuildAwapMessage(AwapMessageTypeEvent, id, seq, payload)
}

// ToJSON converts AWAP message to JSON bytes
func (m *AwapMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// WithHeader adds a header to the AWAP message
func (m *AwapMessage) WithHeader(key, value string) *AwapMessage {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
	return m
}

// WithFormat sets the message format
func (m *AwapMessage) WithFormat(format AwapMessageFormat) *AwapMessage {
	m.Format = format
	return m
}

