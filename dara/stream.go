package dara

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"
)

// 定义 Event 结构体
type SSEEvent struct {
	ID    *string
	Event *string
	Data  *string
}

// 解析单个事件
func parseEvent(eventLines []string) SSEEvent {
	var event SSEEvent
	var data string
	var id string

	for _, line := range eventLines {
		if strings.HasPrefix(line, "data:") {
			data += strings.TrimPrefix(line, "data:") + "\n"
		} else if strings.HasPrefix(line, "id:") {
			id += strings.TrimPrefix(line, "data:") + "\n"
		}
	}

	event.Data = String(data)
	event.ID = String(id)
	return event
}

func ReadAsBytes(body io.Reader) ([]byte, error) {
	byt, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}
	r, ok := body.(io.ReadCloser)
	if ok {
		r.Close()
	}
	return byt, nil
}

func ReadAsJSON(body io.Reader) (result interface{}, err error) {
	byt, err := ioutil.ReadAll(body)
	if err != nil {
		return
	}
	if string(byt) == "" {
		return
	}
	r, ok := body.(io.ReadCloser)
	if ok {
		r.Close()
	}
	d := json.NewDecoder(bytes.NewReader(byt))
	d.UseNumber()
	err = d.Decode(&result)
	return
}

func ReadAsString(body io.Reader) (string, error) {
	byt, err := ioutil.ReadAll(body)
	if err != nil {
		return "", err
	}
	r, ok := body.(io.ReadCloser)
	if ok {
		r.Close()
	}
	return string(byt), nil
}

func ReadAsSSE(body io.ReadCloser) (<-chan SSEEvent, <-chan error) {
	eventChannel := make(chan SSEEvent)

	// 启动 Goroutine 解析 SSE 数据
	go func() {
		defer body.Close()
		defer close(eventChannel)
		var eventLines []string

		reader := bufio.NewReader(body)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimRight(line, "\n")
			if line == "" {
				if len(eventLines) > 0 {
					event := parseEvent(eventLines)
					eventChannel <- event
					eventLines = []string{}
				}
				continue
			}
			eventLines = append(eventLines, line)
		}
	}()
	return eventChannel, nil
}
