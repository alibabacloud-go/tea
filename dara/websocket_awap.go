package dara

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type AwapMessageType string

const (
	// Upstream event types (client -> server)
	AwapMessageTypeUpstreamTextEvent    AwapMessageType = "UpstreamTextEvent"
	AwapMessageTypeUpstreamBinaryEvent  AwapMessageType = "UpstreamBinaryEvent"
	AwapMessageTypeAckRequiredTextEvent AwapMessageType = "AckRequiredTextEvent"

	// Downstream event types (server -> client)
	AwapMessageTypeMessageReceiveEvent   AwapMessageType = "MessageReceiveEvent"
	AwapMessageTypeDownstreamTextEvent   AwapMessageType = "DownstreamTextEvent"
	AwapMessageTypeDownstreamBinaryEvent AwapMessageType = "DownstreamBinaryEvent"

	// Control message types (server -> client)
	AwapMessageTypeReconnect AwapMessageType = "RECONNECT" // Server-initiated graceful reconnection
)

type AwapMessageFormat string

const (
	AwapMessageFormatText   AwapMessageFormat = "text"
	AwapMessageFormatBinary AwapMessageFormat = "binary"
)

type AwapMessage struct {
	Type    AwapMessageType        `json:"type"`
	ID      string                 `json:"id"`
	Seq     int64                  `json:"seq"`
	Headers map[string]string      `json:"headers,omitempty"`
	Payload interface{}            `json:"payload,omitempty"`
	Format  AwapMessageFormat      `json:"format,omitempty"`
	Status  int                    `json:"status,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

type AwapIncomingMessage struct {
	AwapMessage
	RawPayload []byte
}

type AwapWebSocketHandler interface {
	WebSocketHandler

	HandleAwapMessage(session *WebSocketSessionInfo, message *AwapMessage) error

	HandleAwapIncomingMessage(session *WebSocketSessionInfo, message *AwapIncomingMessage) error
}

type AbstractAwapWebSocketHandler struct {
	supportsPartial bool
}

func (h *AbstractAwapWebSocketHandler) AfterConnectionEstablished(session *WebSocketSessionInfo) error {
	return nil
}

func (h *AbstractAwapWebSocketHandler) HandleRawMessage(session *WebSocketSessionInfo, message *WebSocketMessage) error {
	awapMsg, err := ParseAwapMessage(message)
	if err != nil {
		fmt.Printf("[AWAP] Failed to parse message: %v\n", err)
		return err
	}

	fmt.Printf("[tea AWAP] Received message: type=%s, id=%s, seq=%d\n", awapMsg.Type, awapMsg.ID, awapMsg.Seq)

	handler, ok := interface{}(h).(AwapWebSocketHandler)
	if !ok {
		fmt.Printf("[AWAP] ERROR: handler does not implement AwapWebSocketHandler interface, type=%T\n", h)
		return fmt.Errorf("handler does not implement AwapWebSocketHandler")
	}

	fmt.Printf("[AWAP] Handler type: %T, interface handler type: %T\n", h, handler)
	fmt.Printf("[AWAP] Calling HandleAwapMessage for type=%s\n", awapMsg.Type)

	// before calling HandleRawMessage, so it won't reach here. This check is kept for completeness.
	if awapMsg.Type == AwapMessageTypeReconnect {
		fmt.Printf("[AWAP] RECONNECT control message detected (should have been handled earlier)\n")
	}

	// Note: This will call the default implementation if handler is *AbstractAwapWebSocketHandler
	if err := handler.HandleAwapMessage(session, awapMsg); err != nil {
		fmt.Printf("[AWAP] HandleAwapMessage error: %v\n", err)
		return err
	}

	fmt.Printf("[AWAP] HandleAwapMessage completed successfully\n")

	if hasCustomHandleAwapIncomingMessage(handler) {
		incoming := &AwapIncomingMessage{
			AwapMessage: *awapMsg,
			RawPayload:  message.Payload,
		}
		fmt.Printf("[AWAP] Calling HandleAwapIncomingMessage for type=%s\n", awapMsg.Type)
		// Don't return error from HandleAwapIncomingMessage, as HandleAwapMessage already processed it
		if err := handler.HandleAwapIncomingMessage(session, incoming); err != nil {
			fmt.Printf("[AWAP] HandleAwapIncomingMessage error: %v\n", err)
		}
	} else {
		fmt.Printf("[AWAP] HandleAwapIncomingMessage not implemented, skipping\n")
	}

	return nil
}

func (h *AbstractAwapWebSocketHandler) HandleAwapMessage(session *WebSocketSessionInfo, message *AwapMessage) error {
	// Default implementation - can be overridden
	return nil
}

func (h *AbstractAwapWebSocketHandler) HandleAwapIncomingMessage(session *WebSocketSessionInfo, message *AwapIncomingMessage) error {
	// Default implementation - can be overridden
	return nil
}

func hasCustomHandleAwapIncomingMessage(handler AwapWebSocketHandler) bool {
	handlerType := reflect.TypeOf(handler)
	if handlerType == nil {
		return false
	}

	if handlerType.Kind() == reflect.Ptr {
		handlerType = handlerType.Elem()
	}

	// If the handler is AbstractAwapWebSocketHandler, it's using default implementation
	if handlerType == reflect.TypeOf((*AbstractAwapWebSocketHandler)(nil)).Elem() {
		return false
	}

	// Check if the method is from AbstractAwapWebSocketHandler by comparing function addresses
	// Create an instance to get the method value
	defaultHandler := &AbstractAwapWebSocketHandler{}
	defaultMethodValue := reflect.ValueOf(defaultHandler).MethodByName("HandleAwapIncomingMessage")

	handlerValue := reflect.ValueOf(handler)
	if handlerValue.Kind() == reflect.Ptr && handlerValue.IsNil() {
		return false
	}
	handlerMethodValue := handlerValue.MethodByName("HandleAwapIncomingMessage")

	if !handlerMethodValue.IsValid() {
		return false
	}

	if handlerMethodValue.Pointer() == defaultMethodValue.Pointer() {
		return false
	}

	return true
}

func (h *AbstractAwapWebSocketHandler) HandleError(session *WebSocketSessionInfo, err error) error {
	return nil
}

func (h *AbstractAwapWebSocketHandler) AfterConnectionClosed(session *WebSocketSessionInfo, code int, reason string) error {
	return nil
}

func (h *AbstractAwapWebSocketHandler) SupportsPartialMessages() bool {
	return h.supportsPartial
}

func (h *AbstractAwapWebSocketHandler) SetSupportsPartialMessages(supports bool) {
	h.supportsPartial = supports
}

// ParseAwapMessage parses a WebSocket message as AWAP format
// AWAP protocol uses frame format: text headers + JSON payload
// Format: "type:request\nseq:1\ntimestamp:1234567890\nid:msg-001\n\n{JSON payload}"
func ParseAwapMessage(message *WebSocketMessage) (*AwapMessage, error) {
	if message.Type != WebSocketMessageTypeText {
		return nil, fmt.Errorf("AWAP messages must be text format")
	}

	data := message.Payload

	// Check if it's frame format (has \n\n separator)
	headerEndIndex := -1
	for i := 0; i < len(data)-1; i++ {
		if data[i] == '\n' && data[i+1] == '\n' {
			headerEndIndex = i
			break
		}
	}

	awapMsg := &AwapMessage{
		Headers: make(map[string]string),
	}

	if headerEndIndex != -1 {
		// Frame format: parse headers and payload separately
		headerBytes := data[:headerEndIndex]
		payloadBytes := data[headerEndIndex+2:] // Skip \n\n

		// Parse headers (key:value format)
		headerStr := string(headerBytes)
		headerLines := strings.Split(headerStr, "\n")
		for _, line := range headerLines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			colonIndex := strings.Index(line, ":")
			if colonIndex > 0 {
				key := strings.TrimSpace(line[:colonIndex])
				value := strings.TrimSpace(line[colonIndex+1:])
				awapMsg.Headers[key] = value

				// Extract AWAP-specific fields
				switch key {
				case "type":
					awapMsg.Type = AwapMessageType(value)
				case "id":
					awapMsg.ID = value
				case "seq":
					if seq, err := strconv.ParseInt(value, 10, 64); err == nil {
						awapMsg.Seq = seq
					}
				case "status":
					if status, err := strconv.Atoi(value); err == nil {
						awapMsg.Status = status
					}
				case "error":
					awapMsg.Error = value
				}
			}
		}

		// Parse payload as JSON
		if len(payloadBytes) > 0 {
			var payload interface{}
			if err := json.Unmarshal(payloadBytes, &payload); err != nil {
				// If payload is not JSON, treat as raw string
				awapMsg.Payload = string(payloadBytes)
			} else {
				awapMsg.Payload = payload
			}
		}
	} else {
		// Pure JSON format
		if err := json.Unmarshal(data, &awapMsg); err != nil {
			return nil, fmt.Errorf("failed to parse AWAP message: %w", err)
		}
	}

	return awapMsg, nil
}

func (m *AwapMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func (m *AwapMessage) WithHeader(key, value string) *AwapMessage {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
	return m
}

func (m *AwapMessage) WithFormat(format AwapMessageFormat) *AwapMessage {
	m.Format = format
	return m
}
