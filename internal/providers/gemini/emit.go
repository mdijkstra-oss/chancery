package gemini

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
	"hermes-logos/internal/protocol"
)

type SSEEvent struct {
	Type string
	Data string
}

type EmitState struct {
	OutputIndex   int
	NextCallID    int
	ThoughtText   string
	ThoughtSig    []byte
	HasThought    bool
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

	callID := generateCallID(state)
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

func generateCallID(state *EmitState) string {
	id := fmt.Sprintf("call_%d", state.NextCallID)
	state.NextCallID++
	return id
}

func ExtractGeminiUsage(chunk *genai.GenerateContentResponse) *protocol.UsageResponse {
	if chunk == nil || chunk.UsageMetadata == nil {
		return nil
	}
	m := chunk.UsageMetadata
	usage := &protocol.UsageResponse{
		InputTokens:  int(m.PromptTokenCount),
		OutputTokens: int(m.CandidatesTokenCount),
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
			"usage": usage,
		},
	})
	return SSEEvent{
		Type: "response.completed",
		Data: string(data),
	}
}
