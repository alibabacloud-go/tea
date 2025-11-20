package dara

import (
	"encoding/json"
	"fmt"
	"reflect"
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
	Body interface{} `json:"body,omitempty"`
}

type GeneralIncomingMessage struct {
	Body       interface{}
	RawPayload []byte
	IsBinary   bool
}

type GeneralWebSocketHandler interface {
	WebSocketHandler

	HandleGeneralTextMessage(session *WebSocketSessionInfo, message *GeneralMessage) error

	HandleGeneralBinaryMessage(session *WebSocketSessionInfo, data []byte) error

	HandleGeneralIncomingMessage(session *WebSocketSessionInfo, message *GeneralIncomingMessage) error
}

type AbstractGeneralWebSocketHandler struct {
	supportsPartial bool
}

func (h *AbstractGeneralWebSocketHandler) AfterConnectionEstablished(session *WebSocketSessionInfo) error {
	return nil
}

func (h *AbstractGeneralWebSocketHandler) HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error {
	// This method is only called if:
	// 1. DefaultWebSocketClient.readMessages() doesn't recognize the handler as GeneralWebSocketHandler, OR
	// 2. User explicitly overrides this method for custom handling
	//
	// In normal General protocol usage, readMessages() will directly call HandleGeneralTextMessage/HandleGeneralBinaryMessage,
	// so this default implementation won't be called.
	//
	// If you need custom protocol handling, override this method in your handler.
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

func hasCustomHandleGeneralIncomingMessage(handler GeneralWebSocketHandler) bool {
	handlerType := reflect.TypeOf(handler)
	if handlerType == nil {
		return false
	}

	if handlerType.Kind() == reflect.Ptr {
		handlerType = handlerType.Elem()
	}

	if handlerType == reflect.TypeOf((*AbstractGeneralWebSocketHandler)(nil)).Elem() {
		return false
	}

	defaultHandler := &AbstractGeneralWebSocketHandler{}
	defaultMethodValue := reflect.ValueOf(defaultHandler).MethodByName("HandleGeneralIncomingMessage")

	handlerValue := reflect.ValueOf(handler)
	if handlerValue.Kind() == reflect.Ptr && handlerValue.IsNil() {
		return false
	}
	handlerMethodValue := handlerValue.MethodByName("HandleGeneralIncomingMessage")

	if !handlerMethodValue.IsValid() {
		return false
	}

	if handlerMethodValue.Pointer() == defaultMethodValue.Pointer() {
		return false
	}

	return true
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
			Body: string(message.Payload),
		}, nil
	}

	return &generalMsg, nil
}

func (m *GeneralMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
