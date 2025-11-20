package dara

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type AwapMessageType string
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
	// This method is only called if:
	// 1. DefaultWebSocketClient.readMessages() doesn't recognize the handler as AwapWebSocketHandler, OR
	// 2. User explicitly overrides this method for custom handling
	//
	// In normal AWAP protocol usage, readMessages() will directly call HandleAwapMessage,
	// so this default implementation won't be called.
	//
	// If you need custom protocol handling, override this method in your handler.
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
		// First parse as map to check if it matches AWAP structure
		var jsonData map[string]interface{}
		if err := json.Unmarshal(data, &jsonData); err != nil {
			return nil, fmt.Errorf("failed to parse AWAP message as JSON: %w", err)
		}

		// Check if the JSON contains AWAP-specific fields (id, seq, payload, headers)
		// Standard AWAP messages should have at least one of these fields
		_, hasID := jsonData["id"]
		_, hasSeq := jsonData["seq"]
		_, hasPayload := jsonData["payload"]
		_, hasHeaders := jsonData["headers"]
		hasAwapFields := hasID || hasSeq || hasPayload || hasHeaders

		if hasAwapFields {
			// Try to unmarshal as AwapMessage structure
			if err := json.Unmarshal(data, &awapMsg); err != nil {
				return nil, fmt.Errorf("failed to parse AWAP message: %w", err)
			}
		} else {
			// The JSON doesn't match AwapMessage structure (e.g., has receiveTime, clientPayload, type fields)
			// Put the entire JSON object in Data field
			awapMsg.Data = jsonData
			// Also try to extract type if it exists in the JSON (might be MessageReceiveEvent, etc.)
			if msgType, ok := jsonData["type"].(string); ok {
				awapMsg.Type = AwapMessageType(msgType)
			}
			// Initialize Headers if needed
			if awapMsg.Headers == nil {
				awapMsg.Headers = make(map[string]string)
			}
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
