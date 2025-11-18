package dara

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type AwapMessageType string

const (
	// Generic types (for internal use)
	AwapMessageTypeRequest  AwapMessageType = "request"
	AwapMessageTypeResponse AwapMessageType = "response"
	AwapMessageTypeEvent    AwapMessageType = "event"

	// Upstream event types (client -> server)
	AwapMessageTypeUpstreamTextEvent    AwapMessageType = "UpstreamTextEvent"
	AwapMessageTypeUpstreamBinaryEvent  AwapMessageType = "UpstreamBinaryEvent"
	AwapMessageTypeAckRequiredTextEvent AwapMessageType = "AckRequiredTextEvent"

	// Downstream event types (server -> client)
	AwapMessageTypeMessageReceiveEvent   AwapMessageType = "MessageReceiveEvent"
	AwapMessageTypeDownstreamTextEvent   AwapMessageType = "DownstreamTextEvent"
	AwapMessageTypeDownstreamBinaryEvent AwapMessageType = "DownstreamBinaryEvent"
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

	// HandleAwapMessage handles AWAP protocol messages
	HandleAwapMessage(session *WebSocketSessionInfo, message *AwapMessage) error

	// HandleAwapIncomingMessage handles incoming AWAP messages
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

	// Debug: log received message type
	fmt.Printf("[AWAP] Received message: type=%s, id=%s, seq=%d\n", awapMsg.Type, awapMsg.ID, awapMsg.Seq)

	// All AWAP messages should be handled through HandleAwapMessage first
	// This includes both upstream and downstream messages
	// Then, if it's an event type, also call HandleAwapIncomingMessage

	// The key issue: h is *AbstractAwapWebSocketHandler, but we need to call the overridden method.
	// When SequentialHandler.HandleRawMessage calls h.AbstractAwapWebSocketHandler.HandleRawMessage,
	// the receiver in this method is *AbstractAwapWebSocketHandler, not *SequentialHandler.
	//
	// Solution: Since SequentialHandler now overrides HandleRawMessage and calls HandleAwapMessage
	// directly, this code path should only be called for handlers that don't override HandleRawMessage.
	// For those cases, we'll try to call through the interface, but it may still call the default implementation.

	// Try interface assertion
	handler, ok := interface{}(h).(AwapWebSocketHandler)
	if !ok {
		fmt.Printf("[AWAP] ERROR: handler does not implement AwapWebSocketHandler interface, type=%T\n", h)
		return fmt.Errorf("handler does not implement AwapWebSocketHandler")
	}

	fmt.Printf("[AWAP] Handler type: %T, interface handler type: %T\n", h, handler)
	fmt.Printf("[AWAP] Calling HandleAwapMessage for type=%s\n", awapMsg.Type)

	// Call HandleAwapMessage through the interface
	// Note: This will call the default implementation if handler is *AbstractAwapWebSocketHandler
	if err := handler.HandleAwapMessage(session, awapMsg); err != nil {
		fmt.Printf("[AWAP] HandleAwapMessage error: %v\n", err)
		return err
	}

	fmt.Printf("[AWAP] HandleAwapMessage completed successfully\n")

	// For event types (both upstream and downstream), also call HandleAwapIncomingMessage
	if awapMsg.Type == AwapMessageTypeEvent ||
		awapMsg.Type == AwapMessageTypeUpstreamTextEvent ||
		awapMsg.Type == AwapMessageTypeUpstreamBinaryEvent ||
		awapMsg.Type == AwapMessageTypeAckRequiredTextEvent ||
		awapMsg.Type == AwapMessageTypeMessageReceiveEvent ||
		awapMsg.Type == AwapMessageTypeDownstreamTextEvent ||
		awapMsg.Type == AwapMessageTypeDownstreamBinaryEvent {
		incoming := &AwapIncomingMessage{
			AwapMessage: *awapMsg,
			RawPayload:  message.Payload,
		}
		fmt.Printf("[AWAP] Calling HandleAwapIncomingMessage for type=%s\n", awapMsg.Type)
		// Don't return error from HandleAwapIncomingMessage, as HandleAwapMessage already processed it
		if err := handler.HandleAwapIncomingMessage(session, incoming); err != nil {
			fmt.Printf("[AWAP] HandleAwapIncomingMessage error: %v\n", err)
			// Continue anyway, as the message was already handled by HandleAwapMessage
		}
	}

	return nil
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
		// Pure JSON format (backward compatibility)
		if err := json.Unmarshal(data, &awapMsg); err != nil {
			return nil, fmt.Errorf("failed to parse AWAP message: %w", err)
		}
	}

	return awapMsg, nil
}

func BuildAwapMessage(msgType AwapMessageType, id string, seq int64, payload interface{}) *AwapMessage {
	return &AwapMessage{
		Type:    msgType,
		ID:      id,
		Seq:     seq,
		Payload: payload,
		Headers: make(map[string]string),
	}
}

func BuildAwapRequest(id string, seq int64, payload interface{}) *AwapMessage {
	// Use UpstreamTextEvent for text requests (as expected by server)
	return BuildAwapMessage(AwapMessageTypeUpstreamTextEvent, id, seq, payload)
}

func BuildAwapResponse(id string, seq int64, status int, data interface{}) *AwapMessage {
	msg := BuildAwapMessage(AwapMessageTypeResponse, id, seq, nil)
	msg.Status = status
	msg.Data = map[string]interface{}{"result": data}
	return msg
}

func BuildAwapEvent(id string, seq int64, payload interface{}) *AwapMessage {
	// Use UpstreamTextEvent for text events (as expected by server)
	return BuildAwapMessage(AwapMessageTypeUpstreamTextEvent, id, seq, payload)
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
