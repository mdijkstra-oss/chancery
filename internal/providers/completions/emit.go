package completions

import (
	"encoding/json"
	"fmt"

	"hermes-logos/internal/protocol"
	"hermes-logos/internal/providers/sse"
)

type EmitState struct {
	OutputIndex   int
	ActiveCalls   map[int]*activeCall
	ReasoningText string
}

type activeCall struct {
	ID        string
	Name      string
	Arguments string
}

type chunkEnvelope struct {
	Choices []chunkChoice `json:"choices"`
	Usage   *chunkUsage   `json:"usage"`
}

type chunkChoice struct {
	Delta        chunkDelta `json:"delta"`
	FinishReason *string    `json:"finish_reason"`
}

type chunkDelta struct {
	ReasoningContent *string         `json:"reasoning_content"`
	Content          *string         `json:"content"`
	ToolCalls        []toolCallDelta `json:"tool_calls"`
}

type toolCallDelta struct {
	Index    int              `json:"index"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function *toolCallFnDelta `json:"function,omitempty"`
}

type toolCallFnDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type chunkUsage struct {
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	TotalTokens      int           `json:"total_tokens"`
	Details          *usageDetails `json:"prompt_tokens_details,omitempty"`
	CompletionDetail *compDetails  `json:"completion_tokens_details,omitempty"`
}

type usageDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type compDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

func ChunkToEvents(data []byte, state *EmitState) []sse.Event {
	var env chunkEnvelope
	if json.Unmarshal(data, &env) != nil || len(env.Choices) == 0 {
		return nil
	}
	if state.ActiveCalls == nil {
		state.ActiveCalls = make(map[int]*activeCall)
	}
	choice := env.Choices[0]
	var events []sse.Event

	if choice.Delta.ReasoningContent != nil && *choice.Delta.ReasoningContent != "" {
		events = append(events, reasoningDeltaEvent(*choice.Delta.ReasoningContent, state))
	}

	if choice.Delta.Content != nil && *choice.Delta.Content != "" {
		events = append(events, FlushReasoning(state)...)
		events = append(events, sse.TextDeltaEvent(*choice.Delta.Content))
	}

	for _, tc := range choice.Delta.ToolCalls {
		events = append(events, FlushReasoning(state)...)
		events = append(events, toolCallDeltaEvents(tc, state)...)
	}

	if choice.FinishReason != nil {
		events = append(events, FlushReasoning(state)...)
		events = append(events, FlushActiveCalls(state)...)
	}

	return events
}

func reasoningDeltaEvent(text string, state *EmitState) sse.Event {
	state.ReasoningText += text
	return sse.ReasoningDeltaEvent(text)
}

func FlushReasoning(state *EmitState) []sse.Event {
	if state.ReasoningText == "" {
		return nil
	}
	item := map[string]any{
		"type": "reasoning",
		"id":   fmt.Sprintf("rs_%d", state.OutputIndex),
		"summary": []map[string]string{{
			"type": "summary_text",
			"text": state.ReasoningText,
		}},
		"extra_content": map[string]any{
			"deepseek": map[string]string{
				"reasoning_content": state.ReasoningText,
			},
		},
	}
	data, _ := json.Marshal(map[string]any{"item": item})
	state.ReasoningText = ""
	state.OutputIndex++
	return []sse.Event{{
		Type: "response.output_item.done",
		Data: string(data),
	}}
}

func toolCallDeltaEvents(tc toolCallDelta, state *EmitState) []sse.Event {
	var events []sse.Event
	call, exists := state.ActiveCalls[tc.Index]
	if !exists {
		call = &activeCall{}
		state.ActiveCalls[tc.Index] = call
	}

	if tc.ID != "" {
		call.ID = tc.ID
	}
	if tc.Function != nil && tc.Function.Name != "" {
		call.Name = tc.Function.Name
	}

	isNew := tc.ID != ""
	if isNew {
		addedData, _ := json.Marshal(map[string]any{
			"item": map[string]any{
				"type":    "function_call",
				"call_id": call.ID,
				"name":    call.Name,
			},
		})
		events = append(events, sse.Event{
			Type: "response.output_item.added",
			Data: string(addedData),
		})
	}

	if tc.Function != nil && tc.Function.Arguments != "" {
		call.Arguments += tc.Function.Arguments
		deltaData, _ := json.Marshal(map[string]string{"delta": tc.Function.Arguments})
		events = append(events, sse.Event{
			Type: "response.function_call_arguments.delta",
			Data: string(deltaData),
		})
	}

	return events
}

func FlushActiveCalls(state *EmitState) []sse.Event {
	if len(state.ActiveCalls) == 0 {
		return nil
	}
	events := make([]sse.Event, 0, len(state.ActiveCalls))
	for idx := 0; idx < maxActiveIndex(state.ActiveCalls)+1; idx++ {
		call, ok := state.ActiveCalls[idx]
		if !ok {
			continue
		}
		doneData, _ := json.Marshal(map[string]any{
			"item": map[string]any{
				"type":      "function_call",
				"call_id":   call.ID,
				"name":      call.Name,
				"arguments": call.Arguments,
			},
		})
		events = append(events, sse.Event{
			Type: "response.output_item.done",
			Data: string(doneData),
		})
		state.OutputIndex++
	}
	state.ActiveCalls = nil
	return events
}

func maxActiveIndex(calls map[int]*activeCall) int {
	max := 0
	for idx := range calls {
		if idx > max {
			max = idx
		}
	}
	return max
}

func ExtractUsage(data []byte) *protocol.UsageResponse {
	var env chunkEnvelope
	if json.Unmarshal(data, &env) != nil || env.Usage == nil {
		return nil
	}
	u := env.Usage
	usage := &protocol.UsageResponse{
		InputTokens:  u.PromptTokens,
		OutputTokens: u.CompletionTokens,
		TotalTokens:  u.TotalTokens,
	}
	if u.Details != nil && u.Details.CachedTokens > 0 {
		usage.InputTokensDetails = &protocol.PromptTokensDetails{
			CachedTokens: u.Details.CachedTokens,
		}
	}
	if u.CompletionDetail != nil && u.CompletionDetail.ReasoningTokens > 0 {
		usage.OutputTokensDetails = &protocol.OutputTokensDetails{
			ReasoningTokens: u.CompletionDetail.ReasoningTokens,
		}
	}
	return usage
}

func ExtractFinishReason(data []byte) string {
	var env chunkEnvelope
	if json.Unmarshal(data, &env) != nil || len(env.Choices) == 0 {
		return ""
	}
	if env.Choices[0].FinishReason == nil {
		return ""
	}
	return *env.Choices[0].FinishReason
}

var finishReasonErrors = map[string]string{
	"content_filter": "output blocked by content filter",
	"length":         "output truncated: token limit reached",
}

func FinishReasonToEvent(reason string) *sse.Event {
	return sse.FinishReasonToEvent(finishReasonErrors, reason)
}
