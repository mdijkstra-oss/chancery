package gemini

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"

	"google.golang.org/genai"
	"hermes-logos/internal/protocol"
)

type SSEEvent struct {
	Type string
	Data string
}

type EmitState struct {
	OutputIndex int
	ThoughtText string
	ThoughtSig  []byte
	HasThought  bool
}

func ChunkToEvents(chunk *genai.GenerateContentResponse, state *EmitState) []SSEEvent {
	if chunk == nil || len(chunk.Candidates) == 0 || chunk.Candidates[0].Content == nil {
		return nil
	}
	var events []SSEEvent
	for _, part := range chunk.Candidates[0].Content.Parts {
		partEvents := partToEvents(part, state)
		events = append(events, partEvents...)
	}
	return events
}

func partToEvents(part *genai.Part, state *EmitState) []SSEEvent {
	switch {
	case part.FunctionCall != nil:
		return functionCallEvents(part.FunctionCall, part.ThoughtSignature, state)
	case part.Thought:
		return thoughtEvents(part, state)
	case part.Text != "":
		return textEvents(part.Text, state)
	default:
		slog.Warn("gemini: unhandled part type", "has_inline_data", part.InlineData != nil, "has_file_data", part.FileData != nil, "has_code_exec", part.ExecutableCode != nil, "has_code_result", part.CodeExecutionResult != nil)
		return nil
	}
}

func textEvents(text string, state *EmitState) []SSEEvent {
	flushEvents := flushThought(state)
	data, _ := json.Marshal(map[string]string{"delta": text})
	event := SSEEvent{
		Type: "response.output_text.delta",
		Data: string(data),
	}
	return append(flushEvents, event)
}

func functionCallEvents(fc *genai.FunctionCall, sig []byte, state *EmitState) []SSEEvent {
	flushEvents := flushThought(state)

	callID := fc.ID
	argsJSON, _ := json.Marshal(fc.Args)

	addedData, _ := json.Marshal(map[string]any{
		"item": map[string]any{
			"type":    "function_call",
			"call_id": callID,
			"name":    fc.Name,
		},
	})
	added := SSEEvent{
		Type: "response.output_item.added",
		Data: string(addedData),
	}

	deltaData, _ := json.Marshal(map[string]string{"delta": string(argsJSON)})
	delta := SSEEvent{
		Type: "response.function_call_arguments.delta",
		Data: string(deltaData),
	}

	doneItem := map[string]any{
		"type":      "function_call",
		"call_id":   callID,
		"name":      fc.Name,
		"arguments": string(argsJSON),
	}
	if len(sig) > 0 {
		doneItem["extra_content"] = map[string]any{
			"google": map[string]any{
				"thought_signature": base64.StdEncoding.EncodeToString(sig),
			},
		}
	}
	doneData, _ := json.Marshal(map[string]any{"item": doneItem})
	done := SSEEvent{
		Type: "response.output_item.done",
		Data: string(doneData),
	}

	state.OutputIndex++
	return append(flushEvents, added, delta, done)
}

func thoughtEvents(part *genai.Part, state *EmitState) []SSEEvent {
	if part.Text != "" {
		state.ThoughtText += part.Text
		state.HasThought = true
		data, _ := json.Marshal(map[string]string{"delta": part.Text})
		return []SSEEvent{{
			Type: "response.reasoning_summary_text.delta",
			Data: string(data),
		}}
	}
	if len(part.ThoughtSignature) > 0 {
		state.ThoughtSig = part.ThoughtSignature
		state.HasThought = true
	}
	return nil
}

func flushThought(state *EmitState) []SSEEvent {
	if !state.HasThought || len(state.ThoughtSig) == 0 {
		state.HasThought = false
		state.ThoughtText = ""
		state.ThoughtSig = nil
		return nil
	}
	encoded := base64.StdEncoding.EncodeToString(state.ThoughtSig)
	item := map[string]any{
		"type": "reasoning",
		"id":   fmt.Sprintf("rs_%d", state.OutputIndex),
		"extra_content": map[string]any{
			"google": map[string]any{
				"thought_signature": encoded,
			},
		},
	}
	data, _ := json.Marshal(map[string]any{"item": item})
	state.OutputIndex++
	state.HasThought = false
	state.ThoughtText = ""
	state.ThoughtSig = nil
	return []SSEEvent{{
		Type: "response.output_item.done",
		Data: string(data),
	}}
}

func ExtractGeminiUsage(chunk *genai.GenerateContentResponse) *protocol.UsageResponse {
	if chunk == nil || chunk.UsageMetadata == nil {
		return nil
	}
	m := chunk.UsageMetadata
	usage := &protocol.UsageResponse{
		InputTokens:  int(m.PromptTokenCount),
		OutputTokens: int(m.CandidatesTokenCount + m.ThoughtsTokenCount),
		TotalTokens:  int(m.TotalTokenCount),
	}
	if m.CachedContentTokenCount > 0 {
		usage.InputTokensDetails = &protocol.PromptTokensDetails{
			CachedTokens: int(m.CachedContentTokenCount),
		}
	}
	if m.ThoughtsTokenCount > 0 {
		usage.OutputTokensDetails = &protocol.OutputTokensDetails{
			ReasoningTokens: int(m.ThoughtsTokenCount),
		}
	}
	return usage
}

func BuildCompletedEvent(usage *protocol.UsageResponse) SSEEvent {
	data, _ := json.Marshal(map[string]any{
		"response": map[string]any{
			"status": "completed",
			"usage":  usage,
		},
	})
	return SSEEvent{
		Type: "response.completed",
		Data: string(data),
	}
}

func BuildFailedEvent(errorType, message string) SSEEvent {
	data, _ := json.Marshal(map[string]any{
		"response": map[string]any{
			"status": "failed",
			"error": map[string]any{
				"type":    errorType,
				"message": message,
			},
		},
	})
	return SSEEvent{
		Type: "response.failed",
		Data: string(data),
	}
}

func BuildTextDeltaEvent(text string) SSEEvent {
	data, _ := json.Marshal(map[string]string{"delta": text})
	return SSEEvent{
		Type: "response.output_text.delta",
		Data: string(data),
	}
}

func ExtractFinishReason(chunk *genai.GenerateContentResponse) genai.FinishReason {
	if chunk == nil || len(chunk.Candidates) == 0 {
		return ""
	}
	return chunk.Candidates[0].FinishReason
}

var finishReasonErrors = map[genai.FinishReason]string{
	genai.FinishReasonMaxTokens:            "output truncated: token limit reached",
	genai.FinishReasonSafety:               "output blocked by safety filter",
	genai.FinishReasonRecitation:            "output blocked by recitation filter",
	genai.FinishReasonMalformedFunctionCall: "malformed function call",
	genai.FinishReasonBlocklist:             "output blocked by blocklist filter",
	genai.FinishReasonProhibitedContent:     "output blocked: prohibited content",
	genai.FinishReasonSPII:                  "output blocked: sensitive personal information detected",
}

func FinishReasonToEvent(reason genai.FinishReason) *SSEEvent {
	message, ok := finishReasonErrors[reason]
	if !ok {
		return nil
	}
	event := BuildFailedEvent(string(reason), message)
	return &event
}

func ExtractPromptFeedback(chunk *genai.GenerateContentResponse) string {
	if chunk == nil || chunk.PromptFeedback == nil || chunk.PromptFeedback.BlockReason == "" {
		return ""
	}
	return fmt.Sprintf("I'm unable to process this request (blocked: %s).", chunk.PromptFeedback.BlockReason)
}
