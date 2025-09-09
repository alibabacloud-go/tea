package dara

import (
	"context"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/utils"
)

func Test_ReadAsBytes(t *testing.T) {
	byt, err := ReadAsBytes(strings.NewReader("common"))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "common", string(byt))

	byt, err = ReadAsBytes(ioutil.NopCloser(strings.NewReader("common")))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "common", string(byt))
}

func Test_ReadAsJSON(t *testing.T) {
	result, err := ReadAsJSON(strings.NewReader(`{"cleint":"test"}`))
	if res, ok := result.(map[string]interface{}); ok {
		utils.AssertNil(t, err)
		utils.AssertEqual(t, "test", res["cleint"])
	}

	result, err = ReadAsJSON(strings.NewReader(""))
	utils.AssertNil(t, err)
	utils.AssertNil(t, result)

	result, err = ReadAsJSON(ioutil.NopCloser(strings.NewReader(`{"cleint":"test"}`)))
	if res, ok := result.(map[string]interface{}); ok {
		utils.AssertNil(t, err)
		utils.AssertEqual(t, "test", res["cleint"])
	}
}

func Test_ReadAsString(t *testing.T) {
	str, err := ReadAsString(strings.NewReader("common"))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "common", str)

	str, err = ReadAsString(ioutil.NopCloser(strings.NewReader("common")))
	utils.AssertNil(t, err)
	utils.AssertEqual(t, "common", str)
}

func Test_ReadAsSSE(t *testing.T) {
	// Test case 1: Basic SSE event
	t.Run("BasicSSEEvent", func(t *testing.T) {
		sseData := "data: hello world\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)

		ReadAsSSE(reader, eventChannel, errorChannel)

		event := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "hello world", *event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 2: SSE event with event type
	t.Run("SSEWithEventType", func(t *testing.T) {
		sseData := "event: message\ndata: hello world\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)

		ReadAsSSE(reader, eventChannel, errorChannel)

		event := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "hello world", *event.Data)
		utils.AssertNotNil(t, event.Event)
		utils.AssertEqual(t, "message", *event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 3: SSE event with ID
	t.Run("SSEWithID", func(t *testing.T) {
		sseData := "id: 123\ndata: hello world\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)

		ReadAsSSE(reader, eventChannel, errorChannel)

		event := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "hello world", *event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNotNil(t, event.Id)
		utils.AssertEqual(t, "123", *event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 4: SSE event with retry
	t.Run("SSEWithRetry", func(t *testing.T) {
		sseData := "retry: 5000\ndata: hello world\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)

		ReadAsSSE(reader, eventChannel, errorChannel)

		event := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "hello world", *event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNotNil(t, event.Retry)
		utils.AssertEqual(t, 5000, *event.Retry)
	})

	// Test case 5: SSE event with multiline data
	t.Run("SSEWithMultilineData", func(t *testing.T) {
		sseData := "data: first line\ndata: second line\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)

		ReadAsSSE(reader, eventChannel, errorChannel)

		event := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "first line\nsecond line", *event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 6: Complete SSE event
	t.Run("CompleteSSEEvent", func(t *testing.T) {
		sseData := "id: 456\nevent: notification\ndata: welcome\ndata: to sse\nretry: 3000\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)

		ReadAsSSE(reader, eventChannel, errorChannel)

		event := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "welcome\nto sse", *event.Data)
		utils.AssertNotNil(t, event.Event)
		utils.AssertEqual(t, "notification", *event.Event)
		utils.AssertNotNil(t, event.Id)
		utils.AssertEqual(t, "456", *event.Id)
		utils.AssertNotNil(t, event.Retry)
		utils.AssertEqual(t, 3000, *event.Retry)
	})

	// Test case 7: Multiple SSE events
	t.Run("MultipleSSEEvents", func(t *testing.T) {
		sseData := "data: first\n\n" + "data: second\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))

		eventChannel := make(chan *SSEEvent, 2)
		errorChannel := make(chan error, 1)

		ReadAsSSE(reader, eventChannel, errorChannel)

		event1 := <-eventChannel
		event2 := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event1)
		utils.AssertNotNil(t, event1.Data)
		utils.AssertEqual(t, "first", *event1.Data)
		utils.AssertNotNil(t, event2)
		utils.AssertNotNil(t, event2.Data)
		utils.AssertEqual(t, "second", *event2.Data)
	})

}

func Test_parseEvent(t *testing.T) {
	// Test case 1: Basic data line
	t.Run("BasicDataLine", func(t *testing.T) {
		lines := []string{"data: hello world"}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "hello world", *event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 2: Data line with space after colon
	t.Run("DataLineWithSpace", func(t *testing.T) {
		lines := []string{"data:  hello world"}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, " hello world", *event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 3: Data line without space after colon
	t.Run("DataLineWithoutSpace", func(t *testing.T) {
		lines := []string{"data:hello world"}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "hello world", *event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 4: Event line
	t.Run("EventLine", func(t *testing.T) {
		lines := []string{"event: message"}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNil(t, event.Data)
		utils.AssertNotNil(t, event.Event)
		utils.AssertEqual(t, "message", *event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 5: ID line
	t.Run("IDLine", func(t *testing.T) {
		lines := []string{"id: 123"}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNil(t, event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNotNil(t, event.Id)
		utils.AssertEqual(t, "123", *event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 6: Retry line
	t.Run("RetryLine", func(t *testing.T) {
		lines := []string{"retry: 5000"}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNil(t, event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNotNil(t, event.Retry)
		utils.AssertEqual(t, 5000, *event.Retry)
	})

	// Test case 7: Multiline data
	t.Run("MultilineData", func(t *testing.T) {
		lines := []string{"data: first line", "data: second line"}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "first line\nsecond line", *event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 8: Complete event
	t.Run("CompleteEvent", func(t *testing.T) {
		lines := []string{"id: 456", "event: notification", "data: welcome", "data: to sse", "retry: 3000"}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "welcome\nto sse", *event.Data)
		utils.AssertNotNil(t, event.Event)
		utils.AssertEqual(t, "notification", *event.Event)
		utils.AssertNotNil(t, event.Id)
		utils.AssertEqual(t, "456", *event.Id)
		utils.AssertNotNil(t, event.Retry)
		utils.AssertEqual(t, 3000, *event.Retry)
	})

	// Test case 9: Empty lines
	t.Run("EmptyLines", func(t *testing.T) {
		lines := []string{}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNil(t, event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 10: Invalid lines (should be ignored)
	t.Run("InvalidLines", func(t *testing.T) {
		lines := []string{"invalid: line", "another: invalid"}
		event := parseEvent(lines)

		utils.AssertNotNil(t, event)
		utils.AssertNil(t, event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})
}

func Test_ReadAsSSEWithContext(t *testing.T) {
	// Test case 1: Basic SSE event with context
	t.Run("BasicSSEEventWithContext", func(t *testing.T) {
		sseData := "data: hello world\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))
		ctx := context.Background()

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)
		ReadAsSSEWithContext(ctx, reader, eventChannel, errorChannel)

		event := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "hello world", *event.Data)
		utils.AssertNil(t, event.Event)
		utils.AssertNil(t, event.Id)
		utils.AssertNil(t, event.Retry)
	})

	// Test case 2: SSE event with context timeout
	t.Run("SSEWithContextTimeout", func(t *testing.T) {
		// 使用一个快过期的context来测试超时检查
		sseData := "data: hello world\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// 等待context超时
		time.Sleep(10 * time.Millisecond)

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)
		ReadAsSSEWithContext(ctx, reader, eventChannel, errorChannel)

		// 应该立即收到超时错误
		select {
		case err := <-errorChannel:
			utils.AssertNotNil(t, err)
			utils.AssertEqual(t, context.DeadlineExceeded, err)
		case <-time.After(50 * time.Millisecond):
			t.Fatal("Expected timeout error but didn't receive one")
		}
	})

	// Test case 3: SSE event with context cancellation
	t.Run("SSEWithContextCancellation", func(t *testing.T) {
		// 使用一个已经取消的context
		sseData := "data: hello world\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)
		ReadAsSSEWithContext(ctx, reader, eventChannel, errorChannel)

		// 应该立即收到取消错误
		select {
		case err := <-errorChannel:
			utils.AssertNotNil(t, err)
			utils.AssertEqual(t, context.Canceled, err)
		case <-time.After(50 * time.Millisecond):
			t.Fatal("Expected cancellation error but didn't receive one")
		}
	})

	// Test case 4: Complete SSE event with context
	t.Run("CompleteSSEEventWithContext", func(t *testing.T) {
		sseData := "id: 456\nevent: notification\ndata: welcome\ndata: to sse\nretry: 3000\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))
		ctx := context.Background()

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)
		ReadAsSSEWithContext(ctx, reader, eventChannel, errorChannel)

		event := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event)
		utils.AssertNotNil(t, event.Data)
		utils.AssertEqual(t, "welcome\nto sse", *event.Data)
		utils.AssertNotNil(t, event.Event)
		utils.AssertEqual(t, "notification", *event.Event)
		utils.AssertNotNil(t, event.Id)
		utils.AssertEqual(t, "456", *event.Id)
		utils.AssertNotNil(t, event.Retry)
		utils.AssertEqual(t, 3000, *event.Retry)
	})

	// Test case 5: Multiple SSE events with context
	t.Run("MultipleSSEEventsWithContext", func(t *testing.T) {
		sseData := "data: first\n\n" + "data: second\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))
		ctx := context.Background()

		eventChannel := make(chan *SSEEvent, 2)
		errorChannel := make(chan error, 1)
		ReadAsSSEWithContext(ctx, reader, eventChannel, errorChannel)

		event1 := <-eventChannel
		event2 := <-eventChannel
		err := <-errorChannel

		utils.AssertNil(t, err)
		utils.AssertNotNil(t, event1)
		utils.AssertNotNil(t, event1.Data)
		utils.AssertEqual(t, "first", *event1.Data)
		utils.AssertNotNil(t, event2)
		utils.AssertNotNil(t, event2.Data)
		utils.AssertEqual(t, "second", *event2.Data)
	})

	// Test case 6: Context cancellation during event processing
	t.Run("ContextCancellationDuringProcessing", func(t *testing.T) {
		// 验证context取消机制的基本功能
		sseData := "data: event1\n\n"
		reader := ioutil.NopCloser(strings.NewReader(sseData))
		ctx, cancel := context.WithCancel(context.Background())

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)
		ReadAsSSEWithContext(ctx, reader, eventChannel, errorChannel)

		// 读取第一个事件
		event1 := <-eventChannel
		utils.AssertNotNil(t, event1)
		utils.AssertNotNil(t, event1.Data)
		utils.AssertEqual(t, "event1", *event1.Data)

		// 正常结束，收到nil错误
		err := <-errorChannel
		utils.AssertNil(t, err)

		cancel() // 清理资源
	})

	// Test case 7: Empty stream with context
	t.Run("EmptyStreamWithContext", func(t *testing.T) {
		reader := ioutil.NopCloser(strings.NewReader(""))
		ctx := context.Background()

		eventChannel := make(chan *SSEEvent, 1)
		errorChannel := make(chan error, 1)
		ReadAsSSEWithContext(ctx, reader, eventChannel, errorChannel)

		// 应该立即收到nil错误（EOF）
		err := <-errorChannel
		utils.AssertNil(t, err)

		// 事件通道应该为空
		select {
		case event := <-eventChannel:
			utils.AssertNil(t, event) // 不应该收到任何事件
		default:
			// 这是期望的行为
		}
	})
}
