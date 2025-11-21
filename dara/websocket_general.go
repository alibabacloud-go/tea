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

type AbstractGeneralWebSocketHandler struct {
	supportsPartial bool
}

func (h *AbstractGeneralWebSocketHandler) AfterConnectionEstablished(session *WebSocketSessionInfo) error {
	return nil
}

func (h *AbstractGeneralWebSocketHandler) HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error {
	// This method is only called if:
	// 1. DefaultWebSocketClient.readMessages() doesn't recognize the handler as GeneralWebSocketHandler or AwapWebSocketHandler,
	// 2. User explicitly overrides this method for custom handling
	//
	// In normal General protocol usage, readMessages() will directly call HandleGeneralTextMessage/HandleGeneralBinaryMessage,
	// so this default implementation won't be called.
	//
	// If you need custom protocol handling, override this method in your handler.
	return nil
}

func (h *AbstractGeneralWebSocketHandler) HandleGeneralMessage(session *WebSocketSessionInfo, message *GeneralMessage) error {
	// Default implementation returns ErrUseRawMessage to indicate HandleRawMessage should be used
	// If user overrides this method, they should return nil or their own error (not ErrUseRawMessage)
	return ErrUseRawMessage
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
