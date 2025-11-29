package dara

import (
	"encoding/json"
)

type GeneralMessageFormat string

const (
	GeneralMessageFormatText   GeneralMessageFormat = "text"
	GeneralMessageFormatBinary GeneralMessageFormat = "binary"
)

type GeneralMessage struct {
	Body   interface{}          `json:"body,omitempty"`
	Format GeneralMessageFormat `json:"format,omitempty"`
}

type GeneralWebSocketHandler interface {
	WebSocketHandler

	HandleGeneralMessage(session *WebSocketSessionInfo, message *GeneralMessage) error
}

// AbstractGeneralWebSocketHandler provides default implementations for GeneralWebSocketHandler interface
// It embeds AbstractWebSocketHandler for base WebSocketHandler methods
// Users can embed this struct in their custom General handlers
type AbstractGeneralWebSocketHandler struct {
	AbstractWebSocketHandler
}

func (h *AbstractGeneralWebSocketHandler) HandleGeneralMessage(session *WebSocketSessionInfo, message *GeneralMessage) error {
	// Default implementation returns ErrUseRawMessage to indicate HandleRawMessage should be used
	// If user overrides this method, they should return nil or their own error (not ErrUseRawMessage)
	return ErrUseRawMessage
}

func ParseGeneralMessage(message *WebSocketMessage) (*GeneralMessage, error) {
	if message.Type == WebSocketMessageTypeBinary {
		return &GeneralMessage{
			Body:   message.Payload,
			Format: GeneralMessageFormatBinary,
		}, nil
	}
	// Try to parse the entire JSON payload as the body
	// For text messages, payload is already a string (UTF-8 encoded bytes)
	// If JSON parsing fails, return the original bytes as []byte to preserve the data
	var body interface{}
	if err := json.Unmarshal(message.Payload, &body); err != nil {
		return &GeneralMessage{
			Body:   message.Payload,
			Format: GeneralMessageFormatText,
		}, nil
	}
	// Successfully parsed as JSON, return the parsed object
	return &GeneralMessage{
		Body:   body,
		Format: GeneralMessageFormatText,
	}, nil
}

func (m *GeneralMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
