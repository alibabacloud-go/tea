package dara

import (
	"encoding/json"
	"fmt"
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
	// only type and id is required for awap message, and id can be autofilled when sending message
	// other fields are optional and can be set using withHeader
	Type    AwapMessageType        `json:"type"`
	ID      string                 `json:"id"`
	Headers map[string]string      `json:"headers,omitempty"`
	Payload interface{}            `json:"payload,omitempty"`
	Format  AwapMessageFormat      `json:"format,omitempty"`
	Status  int                    `json:"status,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
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

type AwapWebSocketHandler interface {
	WebSocketHandler

	HandleAwapMessage(session *WebSocketSessionInfo, message *AwapMessage) error
}

// AbstractAwapWebSocketHandler provides default implementations for AwapWebSocketHandler interface
// It embeds AbstractWebSocketHandler for base WebSocketHandler methods
// Users can embed this struct in their custom AWAP handlers
type AbstractAwapWebSocketHandler struct {
	AbstractWebSocketHandler
}

// ErrUseRawMessage is a sentinel error that indicates HandleRawMessage should be used instead
var ErrUseRawMessage = fmt.Errorf("use HandleRawMessage")

func (h *AbstractAwapWebSocketHandler) HandleAwapMessage(session *WebSocketSessionInfo, message *AwapMessage) error {
	// Default implementation returns ErrUseRawMessage to indicate HandleRawMessage should be used
	// If user overrides this method, they should return nil or their own error (not ErrUseRawMessage)
	return ErrUseRawMessage
}

// ParseAwapMessage parses a WebSocket message as AWAP format
// AWAP protocol uses frame format: text headers + JSON payload
// Format: "type:request\ntimestamp:1234567890\nid:msg-001\n\n{JSON payload} or [header bytes (text converted to binary)] + [\n\n separator] + [binary body]"
func ParseAwapMessage(message *WebSocketMessage) (*AwapMessage, error) {
	data := message.Payload

	// Check if it's awap format (has \n\n separator)
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

	if headerEndIndex == -1 {
		return nil, fmt.Errorf("failed to parse AWAP message: no \n\n separator found")
	}

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
			// Extract AWAP-required fields
			switch key {
			case "type":
				awapMsg.Type = AwapMessageType(value)
			case "id":
				awapMsg.ID = value
			case "status":
				if status, err := strconv.Atoi(value); err == nil {
					awapMsg.Status = status
				}
			case "error":
				awapMsg.Error = value
			default:
				// Only non-AWAP-required fields go into Headers map
				awapMsg.Headers[key] = value
			}
		}
	}

	if len(payloadBytes) > 0 {
		var payload interface{}
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			awapMsg.Payload = payloadBytes
		} else {
			awapMsg.Payload = payload
		}
	}

	if message.Type == WebSocketMessageTypeBinary {
		awapMsg.Format = AwapMessageFormatBinary
	} else {
		awapMsg.Format = AwapMessageFormatText
	}

	return awapMsg, nil
}
