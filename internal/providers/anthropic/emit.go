package anthropic

import (
	"encoding/json"

	"hermes-logos/internal/protocol"
	"hermes-logos/internal/providers/sse"
)

type EmitState struct {
	OutputIndex     int
	ActiveBlockType string
	ActiveBlockIdx  int
	ThinkingText    string
	ThinkingSig     string
	ToolCallID      string
	ToolCallName    string
	ToolCallJSON    string
	InputTokens     int
	CacheRead       int
	CacheCreation   int
}

type messageStartPayload struct {
	Message struct {
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

type contentBlockStartPayload struct {
	Index        int `json:"index"`
	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"content_block"`
}

type contentBlockDeltaPayload struct {
	Index int `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
		Thinking    string `json:"thinking"`
		Signature   string `json:"signature"`
	} `json:"delta"`
}

type messageDeltaPayload struct {
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage struct {
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
}

type errorPayload struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func HandleEvent(eventType string, data []byte, state *EmitState) []sse.Event {
	switch eventType {
	case "message_start":
		return handleMessageStart(data, state)
	case "content_block_start":
		return handleContentBlockStart(data, state)
	case "content_block_delta":
		return handleContentBlockDelta(data, state)
	case "content_block_stop":
		return handleContentBlockStop(state)
	case "message_delta":
		return handleMessageDelta(data, state)
	case "error":
		return handleError(data)
	case "ping", "message_stop":
		return nil
	default:
		return nil
	}
}

func handleMessageStart(data []byte, state *EmitState) []sse.Event {
	var p messageStartPayload
	if json.Unmarshal(data, &p) != nil {
		return nil
	}
	state.InputTokens = p.Message.Usage.InputTokens
	state.CacheRead = p.Message.Usage.CacheReadInputTokens
	state.CacheCreation = p.Message.Usage.CacheCreationInputTokens
	return nil
}

func handleContentBlockStart(data []byte, state *EmitState) []sse.Event {
	var p contentBlockStartPayload
	if json.Unmarshal(data, &p) != nil {
		return nil
	}
	state.ActiveBlockType = p.ContentBlock.Type
	state.ActiveBlockIdx = p.Index
	switch p.ContentBlock.Type {
	case "tool_use":
		state.ToolCallID = p.ContentBlock.ID
		state.ToolCallName = p.ContentBlock.Name
		state.ToolCallJSON = ""
		return []sse.Event{buildFunctionCallAddedEvent(state.OutputIndex, p.ContentBlock.ID, p.ContentBlock.Name)}
	case "thinking":
		state.ThinkingText = ""
		state.ThinkingSig = ""
		return nil
	default:
		return nil
	}
}

func handleContentBlockDelta(data []byte, state *EmitState) []sse.Event {
	var p contentBlockDeltaPayload
	if json.Unmarshal(data, &p) != nil {
		return nil
	}
	switch p.Delta.Type {
	case "text_delta":
		return []sse.Event{buildTextDeltaEvent(p.Delta.Text)}
	case "input_json_delta":
		state.ToolCallJSON += p.Delta.PartialJSON
		return []sse.Event{buildFunctionCallArgsDeltaEvent(state.OutputIndex, p.Delta.PartialJSON)}
	case "thinking_delta":
		state.ThinkingText += p.Delta.Thinking
		return []sse.Event{buildReasoningDeltaEvent(p.Delta.Thinking)}
	case "signature_delta":
		state.ThinkingSig += p.Delta.Signature
		return nil
	default:
		return nil
	}
}

func handleContentBlockStop(state *EmitState) []sse.Event {
	switch state.ActiveBlockType {
	case "thinking":
		event := buildReasoningDoneEvent(state.OutputIndex, state.ThinkingText, state.ThinkingSig)
		state.OutputIndex++
		state.ThinkingText = ""
		state.ThinkingSig = ""
		return []sse.Event{event}
	case "tool_use":
		event := buildFunctionCallDoneEvent(state.OutputIndex, state.ToolCallID, state.ToolCallName, state.ToolCallJSON)
		state.OutputIndex++
		state.ToolCallID = ""
		state.ToolCallName = ""
		state.ToolCallJSON = ""
		return []sse.Event{event}
	default:
		return nil
	}
}

func handleMessageDelta(data []byte, state *EmitState) []sse.Event {
	var p messageDeltaPayload
	if json.Unmarshal(data, &p) != nil {
		return nil
	}
	if p.Usage.CacheReadInputTokens > 0 {
		state.CacheRead = p.Usage.CacheReadInputTokens
	}
	if p.Usage.CacheCreationInputTokens > 0 {
		state.CacheCreation = p.Usage.CacheCreationInputTokens
	}
	if isStopError(p.Delta.StopReason) {
		return []sse.Event{sse.BuildFailedEvent("max_tokens", "output truncated: token limit reached")}
	}
	return nil
}

func handleError(data []byte) []sse.Event {
	var p errorPayload
	if json.Unmarshal(data, &p) != nil {
		return []sse.Event{sse.BuildFailedEvent("unknown_error", "unknown streaming error")}
	}
	return []sse.Event{sse.BuildFailedEvent(p.Error.Type, p.Error.Message)}
}

func isStopError(reason string) bool {
	return reason == "max_tokens"
}

func ExtractUsage(state *EmitState, finalData []byte) *protocol.UsageResponse {
	var p messageDeltaPayload
	json.Unmarshal(finalData, &p)

	outputTokens := p.Usage.OutputTokens
	cacheRead := state.CacheRead
	cacheCreation := state.CacheCreation
	if p.Usage.CacheReadInputTokens > 0 {
		cacheRead = p.Usage.CacheReadInputTokens
	}
	if p.Usage.CacheCreationInputTokens > 0 {
		cacheCreation = p.Usage.CacheCreationInputTokens
	}

	inputTokens := state.InputTokens + cacheRead + cacheCreation
	totalTokens := inputTokens + outputTokens

	usage := &protocol.UsageResponse{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
	}
	if cacheRead > 0 || cacheCreation > 0 {
		usage.InputTokensDetails = &protocol.PromptTokensDetails{
			CachedTokens:        cacheRead,
			CacheCreationTokens: cacheCreation,
		}
	}
	return usage
}

func buildTextDeltaEvent(text string) sse.Event {
	data, _ := json.Marshal(map[string]string{"delta": text})
	return sse.Event{Type: "response.output_text.delta", Data: string(data)}
}

func buildReasoningDeltaEvent(text string) sse.Event {
	data, _ := json.Marshal(map[string]string{"delta": text})
	return sse.Event{Type: "response.reasoning_summary_text.delta", Data: string(data)}
}

func buildReasoningDoneEvent(outputIndex int, thinking, signature string) sse.Event {
	data, _ := json.Marshal(map[string]any{
		"output_index": outputIndex,
		"item": map[string]any{
			"type": "reasoning",
			"id":   "",
			"extra_content": map[string]any{
				"anthropic": map[string]string{
					"thinking":  thinking,
					"signature": signature,
				},
			},
		},
	})
	return sse.Event{Type: "response.output_item.done", Data: string(data)}
}

func buildFunctionCallAddedEvent(outputIndex int, callID, name string) sse.Event {
	data, _ := json.Marshal(map[string]any{
		"output_index": outputIndex,
		"item": map[string]any{
			"type":    "function_call",
			"call_id": callID,
			"name":    name,
		},
	})
	return sse.Event{Type: "response.output_item.added", Data: string(data)}
}

func buildFunctionCallArgsDeltaEvent(outputIndex int, delta string) sse.Event {
	data, _ := json.Marshal(map[string]any{
		"output_index": outputIndex,
		"delta":        delta,
	})
	return sse.Event{Type: "response.function_call_arguments.delta", Data: string(data)}
}

func buildFunctionCallDoneEvent(outputIndex int, callID, name, arguments string) sse.Event {
	data, _ := json.Marshal(map[string]any{
		"output_index": outputIndex,
		"item": map[string]any{
			"type":      "function_call",
			"call_id":   callID,
			"name":      name,
			"arguments": arguments,
		},
	})
	return sse.Event{Type: "response.output_item.done", Data: string(data)}
}
