package dara

import (
	"encoding/json"
	"fmt"
)

type GeneralMessageType string

const (
	// Upstream event types (client -> server)
	GeneralMessageTypeUpstreamDefaultTextEvent   GeneralMessageType = "UpstreamDefaultTextEvent"
	GeneralMessageTypeUpstreamDefaultBinaryEvent GeneralMessageType = "UpstreamDefaultBinaryEvent"

	// Downstream event types (server -> client)
	GeneralMessageTypeDownstreamDefaultTextEvent   GeneralMessageType = "DownstreamDefaultTextEvent"
	GeneralMessageTypeDownstreamDefaultBinaryEvent GeneralMessageType = "DownstreamDefaultBinaryEvent"
)

type GeneralMessage struct {
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
}

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

type AbstractGeneralWebSocketHandler struct {
	supportsPartial bool
}

func (h *AbstractGeneralWebSocketHandler) AfterConnectionEstablished(session *WebSocketSessionInfo) error {
	return nil
}

func (h *AbstractGeneralWebSocketHandler) HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error {
	fmt.Printf("[General] HandleRawMessage called: type=%d, payloadLen=%d, payload=%s\n",
		message.Type, len(message.Payload), string(message.Payload))

	// In Go, when a struct embeds AbstractGeneralWebSocketHandler and overrides methods,
	// calling h.HandleGeneralTextMessage will correctly call the overridden method.
	// However, we need to ensure we're calling through the interface to get proper dispatch.
	// Convert to interface to ensure we call the correct implementation
	handler := interface{}(h).(GeneralWebSocketHandler)

	if message.Type == WebSocketMessageTypeText {
		generalMsg, err := ParseGeneralMessage(message)
		if err != nil {
			fmt.Printf("[General] Failed to parse General message: %v\n", err)
			return err
		}

		incoming := &GeneralIncomingMessage{
			Headers:    generalMsg.Headers,
			Body:       generalMsg.Body,
			RawPayload: message.Payload,
			IsBinary:   false,
		}

		// Call both handlers using the interface to ensure correct method dispatch
		fmt.Printf("[General] Calling HandleGeneralTextMessage\n")
		if err := handler.HandleGeneralTextMessage(session, generalMsg); err != nil {
			fmt.Printf("[General] HandleGeneralTextMessage error: %v\n", err)
			return err
		}
		fmt.Printf("[General] Calling HandleGeneralIncomingMessage\n")
		return handler.HandleGeneralIncomingMessage(session, incoming)

	} else if message.Type == WebSocketMessageTypeBinary {
		// Handle as binary message
		incoming := &GeneralIncomingMessage{
			Headers:    make(map[string]string),
			Body:       nil,
			RawPayload: message.Payload,
			IsBinary:   true,
		}

		// Call both handlers using the interface to ensure correct method dispatch
		fmt.Printf("[General] Calling HandleGeneralBinaryMessage\n")
		if err := handler.HandleGeneralBinaryMessage(session, message.Payload); err != nil {
			fmt.Printf("[General] HandleGeneralBinaryMessage error: %v\n", err)
			return err
		}
		fmt.Printf("[General] Calling HandleGeneralIncomingMessage\n")
		return handler.HandleGeneralIncomingMessage(session, incoming)
	}

	fmt.Printf("[General] Unknown message type: %d\n", message.Type)
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
		return nil, fmt.Errorf("general text messages must be text format")
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
