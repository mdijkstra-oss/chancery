package sse

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

type Collector struct {
	buf bytes.Buffer
}

type ToolCall struct {
	Name      string
	CallID    string
	Arguments string
}

type CollectedResponse struct {
	Text      string
	ToolCalls []ToolCall
}

func (c *Collector) Write(p []byte) (int, error) {
	return c.buf.Write(p)
}

func (c *Collector) Result() CollectedResponse {
	return ParseSSEOutput(c.buf.Bytes())
}

func ParseSSEOutput(data []byte) CollectedResponse {
	var result CollectedResponse
	var currentEvent string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			currentEvent = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			dataStr := strings.TrimPrefix(line, "data: ")
			switch currentEvent {
			case "response.output_text.delta":
				result.Text += extractDelta(dataStr)
			case "response.output_item.added":
				if call := extractToolCallAdded(dataStr); call != nil {
					result.ToolCalls = append(result.ToolCalls, *call)
				}
			case "response.function_call_arguments.delta":
				if len(result.ToolCalls) > 0 {
					last := &result.ToolCalls[len(result.ToolCalls)-1]
					last.Arguments += extractDelta(dataStr)
				}
			}
		case line == "":
			currentEvent = ""
		}
	}
	return result
}

func extractDelta(data string) string {
	var obj struct {
		Delta string `json:"delta"`
	}
	json.Unmarshal([]byte(data), &obj)
	return obj.Delta
}

func extractToolCallAdded(data string) *ToolCall {
	var obj struct {
		Item struct {
			Type   string `json:"type"`
			CallID string `json:"call_id"`
			Name   string `json:"name"`
		} `json:"item"`
	}
	if json.Unmarshal([]byte(data), &obj) != nil || obj.Item.Type != "function_call" {
		return nil
	}
	return &ToolCall{Name: obj.Item.Name, CallID: obj.Item.CallID}
}
