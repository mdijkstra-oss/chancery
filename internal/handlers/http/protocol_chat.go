package http

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type ChatCompletionsRequest struct {
	Model          string            `json:"model"`
	Messages       []json.RawMessage `json:"messages"`
	Tools          []json.RawMessage `json:"tools,omitempty"`
	ToolChoice     *string           `json:"tool_choice,omitempty"`
	Temperature    *float64          `json:"temperature,omitempty"`
	Stream         bool              `json:"stream"`
	StreamOptions  *StreamOptions    `json:"stream_options,omitempty"`
	ResponseFormat json.RawMessage   `json:"response_format,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content,omitempty"`
	ToolCalls []chatToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type responsesMessage struct {
	Type      string `json:"type"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Output    string `json:"output"`
}

func buildChatRequestFromParams(p RequestParams) any {
	messages := toChatMessages(p.SystemPrompt, p.Messages)
	req := ChatCompletionsRequest{
		Model:         p.Model,
		Messages:      messages,
		Tools:         p.Tools,
		Stream:        true,
		StreamOptions: &StreamOptions{IncludeUsage: true},
	}
	if p.ToolChoice != "" && len(p.Tools) > 0 {
		req.ToolChoice = &p.ToolChoice
	}
	if p.Temperature != nil {
		req.Temperature = p.Temperature
	}
	if p.ResponseFormat != nil {
		req.ResponseFormat = p.ResponseFormat
	}
	return req
}

func toChatMessages(systemPrompt string, input []json.RawMessage) []json.RawMessage {
	result := make([]json.RawMessage, 0, len(input)+1)
	sysMsg, _ := json.Marshal(chatMessage{Role: "system", Content: quotedString(systemPrompt)})
	result = append(result, sysMsg)

	for _, raw := range input {
		var msg responsesMessage
		if json.Unmarshal(raw, &msg) != nil {
			result = append(result, raw)
			continue
		}
		translated := translateMessage(msg)
		if translated != nil {
			result = append(result, translated)
		}
	}
	return result
}

func quotedString(s string) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}

func translateMessage(msg responsesMessage) json.RawMessage {
	switch {
	case isContentMessage(msg):
		return marshalChatMessage(chatMessage{Role: msg.Role, Content: quotedString(msg.Content)})
	case isFunctionCallMessage(msg):
		return marshalChatMessage(chatMessage{
			Role: "assistant",
			ToolCalls: []chatToolCall{{
				ID:   msg.CallID,
				Type: "function",
				Function: chatToolFunction{
					Name:      msg.Name,
					Arguments: msg.Arguments,
				},
			}},
		})
	case isFunctionCallOutputMessage(msg):
		return marshalChatMessage(chatMessage{
			Role:       "tool",
			ToolCallID: msg.CallID,
			Content:    quotedString(msg.Output),
		})
	default:
		return nil
	}
}

func isContentMessage(msg responsesMessage) bool {
	return msg.Type == "message" && msg.Role != ""
}

func isFunctionCallMessage(msg responsesMessage) bool {
	return msg.Type == "function_call"
}

func isFunctionCallOutputMessage(msg responsesMessage) bool {
	return msg.Type == "function_call_output"
}

func marshalChatMessage(m chatMessage) json.RawMessage {
	data, _ := json.Marshal(m)
	return data
}

type chatChunk struct {
	Choices []chatChunkChoice `json:"choices"`
	Usage   *chatChunkUsage   `json:"usage,omitempty"`
}

type chatChunkChoice struct {
	Delta        chatChunkDelta `json:"delta"`
	FinishReason *string        `json:"finish_reason"`
}

type chatChunkDelta struct {
	Content   *string         `json:"content,omitempty"`
	ToolCalls []chatToolDelta `json:"tool_calls,omitempty"`
}

type chatToolDelta struct {
	Index    int               `json:"index"`
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type,omitempty"`
	Function *chatFuncDelta    `json:"function,omitempty"`
}

type chatFuncDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type chatChunkUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func forwardChatStream(ctx context.Context, src io.Reader, dst io.Writer, flusher http.Flusher, inspect bool, endpoint string) *UsageResponse {
	scanner := bufio.NewScanner(src)
	var usage *UsageResponse
	var pendingToolCalls []pendingTool

	for scanner.Scan() {
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		if data == "[DONE]" {
			break
		}

		var chunk chatChunk
		if json.Unmarshal([]byte(data), &chunk) != nil {
			continue
		}

		if chunk.Usage != nil {
			usage = chatUsageToResponses(chunk.Usage)
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		emitContentDelta(dst, flusher, choice.Delta.Content)
		pendingToolCalls = emitToolDeltas(dst, flusher, choice.Delta.ToolCalls, pendingToolCalls)

		if choice.FinishReason != nil {
			emitOutputDone(dst, flusher, pendingToolCalls)
		}
	}

	if err := scanner.Err(); err != nil && isUnexpectedStreamError(err) {
		slog.ErrorContext(ctx, "chat stream read error", "component", "stream", "error", err)
		return nil
	}

	emitCompleted(dst, flusher, usage)
	return usage
}

type pendingTool struct {
	ID   string
	Name string
}

func emitContentDelta(dst io.Writer, flusher http.Flusher, content *string) {
	if content == nil {
		return
	}
	escaped, _ := json.Marshal(*content)
	writeSSEEvent(dst, flusher, "response.output_text.delta",
		fmt.Sprintf(`{"delta":%s}`, string(escaped)))
}

func emitToolDeltas(dst io.Writer, flusher http.Flusher, deltas []chatToolDelta, pending []pendingTool) []pendingTool {
	for _, td := range deltas {
		for td.Index >= len(pending) {
			pending = append(pending, pendingTool{})
		}

		if td.ID != "" {
			pending[td.Index].ID = td.ID
		}
		if td.Function != nil && td.Function.Name != "" {
			pending[td.Index].Name = td.Function.Name
			writeSSEEvent(dst, flusher, "response.output_item.added",
				fmt.Sprintf(`{"item":{"type":"function_call","call_id":%q,"name":%q}}`,
					pending[td.Index].ID, td.Function.Name))
		}
		if td.Function != nil && td.Function.Arguments != "" {
			escaped, _ := json.Marshal(td.Function.Arguments)
			writeSSEEvent(dst, flusher, "response.function_call_arguments.delta",
				fmt.Sprintf(`{"delta":%s}`, string(escaped)))
		}
	}
	return pending
}

func emitOutputDone(dst io.Writer, flusher http.Flusher, pending []pendingTool) {
	for _, tool := range pending {
		if tool.Name == "" {
			continue
		}
		writeSSEEvent(dst, flusher, "response.output_item.done",
			fmt.Sprintf(`{"item":{"type":"function_call","call_id":%q,"name":%q}}`,
				tool.ID, tool.Name))
	}
}

func emitCompleted(dst io.Writer, flusher http.Flusher, usage *UsageResponse) {
	event := ResponseCompletedEvent{
		Type:     "response.completed",
		Response: ResponseObject{Usage: usage},
	}
	data, _ := json.Marshal(event)
	writeSSEEvent(dst, flusher, "response.completed", string(data))
}

func chatUsageToResponses(cu *chatChunkUsage) *UsageResponse {
	return &UsageResponse{
		InputTokens:  cu.PromptTokens,
		OutputTokens: cu.CompletionTokens,
		TotalTokens:  cu.TotalTokens,
	}
}
