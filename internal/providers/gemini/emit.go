package gemini

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/matthijn/hermes-logos/internal/protocol"
	"github.com/matthijn/hermes-logos/internal/providers/sse"
	"google.golang.org/genai"
)

type EmitState struct {
	OutputIndex int
	ThoughtSig  []byte
	HasThought  bool
	Suppressing bool
	LeakedBuf   string
}

func ChunkToEvents(chunk *genai.GenerateContentResponse, state *EmitState) []sse.Event {
	if chunk == nil || len(chunk.Candidates) == 0 || chunk.Candidates[0].Content == nil {
		return nil
	}
	var events []sse.Event
	for _, part := range chunk.Candidates[0].Content.Parts {
		partEvents := partToEvents(part, state)
		events = append(events, partEvents...)
	}
	return events
}

func isEmptyPart(part *genai.Part) bool {
	return part.Text == "" &&
		!part.Thought &&
		part.FunctionCall == nil &&
		part.FunctionResponse == nil &&
		part.InlineData == nil &&
		part.FileData == nil &&
		part.ExecutableCode == nil &&
		part.CodeExecutionResult == nil &&
		len(part.ThoughtSignature) == 0
}

func partToEvents(part *genai.Part, state *EmitState) []sse.Event {
	switch {
	case part.FunctionCall != nil:
		return functionCallEvents(part.FunctionCall, part.ThoughtSignature, state)
	case part.Thought:
		return thoughtEvents(part, state)
	case part.Text != "":
		events := textEvents(part.Text, state)
		if len(part.ThoughtSignature) > 0 {
			state.ThoughtSig = part.ThoughtSignature
			state.HasThought = true
		}
		return events
	case len(part.ThoughtSignature) > 0:
		return thoughtEvents(part, state)
	case isEmptyPart(part):
		return nil
	default:
		raw, _ := json.Marshal(part)
		slog.Warn("gemini: unhandled part type", "part", string(raw))
		return nil
	}
}

type textSegment struct {
	content     string
	isReasoning bool
}

var leakedOpenTags = []string{"<thinking>", "<thought>", "<think>"}
var leakedCloseTags = []string{"</thinking>", "</thought>", "</think>"}

func textEvents(text string, state *EmitState) []sse.Event {
	segments := filterLeakedThinking(text, state)
	var events []sse.Event
	for _, seg := range segments {
		if seg.isReasoning {
			events = append(events, emitReasoningDelta(seg.content, state)...)
		} else {
			events = append(events, emitTextDelta(seg.content, state)...)
		}
	}
	return events
}

func emitTextDelta(text string, state *EmitState) []sse.Event {
	flushEvents := flushThought(state)
	return append(flushEvents, sse.TextDeltaEvent(text))
}

func emitReasoningDelta(text string, state *EmitState) []sse.Event {
	state.HasThought = true
	return []sse.Event{sse.ReasoningDeltaEvent(text)}
}

func filterLeakedThinking(text string, state *EmitState) []textSegment {
	input := state.LeakedBuf + text
	state.LeakedBuf = ""

	if len(input) == 0 {
		return nil
	}

	var segments []textSegment

	for len(input) > 0 {
		tags, isReasoning := activeLeakedTags(state.Suppressing)
		idx, tagLen := indexOfAny(input, tags)
		if idx == -1 {
			partial := longestPartialSuffix(input, tags)
			if partial > 0 && partial < len(input) {
				segments = append(segments, textSegment{input[:len(input)-partial], isReasoning})
				state.LeakedBuf = input[len(input)-partial:]
			} else if partial == len(input) {
				state.LeakedBuf = input
			} else {
				segments = append(segments, textSegment{input, isReasoning})
			}
			break
		}
		if idx > 0 {
			segments = append(segments, textSegment{input[:idx], isReasoning})
		}
		input = input[idx+tagLen:]
		state.Suppressing = !state.Suppressing
	}

	return segments
}

func activeLeakedTags(suppressing bool) ([]string, bool) {
	if suppressing {
		return leakedCloseTags, true
	}
	return leakedOpenTags, false
}

func indexOfAny(s string, needles []string) (int, int) {
	bestIdx := -1
	bestLen := 0
	for _, needle := range needles {
		idx := strings.Index(s, needle)
		if idx != -1 && (bestIdx == -1 || idx < bestIdx || (idx == bestIdx && len(needle) > bestLen)) {
			bestIdx = idx
			bestLen = len(needle)
		}
	}
	return bestIdx, bestLen
}

func longestPartialSuffix(s string, tags []string) int {
	best := 0
	for _, tag := range tags {
		maxN := len(tag) - 1
		if maxN > len(s) {
			maxN = len(s)
		}
		for n := maxN; n >= 1; n-- {
			if s[len(s)-n:] == tag[:n] {
				if n > best {
					best = n
				}
				break
			}
		}
	}
	return best
}

func functionCallEvents(fc *genai.FunctionCall, sig []byte, state *EmitState) []sse.Event {
	if len(sig) == 0 && len(state.ThoughtSig) > 0 {
		sig = state.ThoughtSig
		state.ThoughtSig = nil
	}
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
	added := sse.Event{
		Type: "response.output_item.added",
		Data: string(addedData),
	}

	deltaData, _ := json.Marshal(map[string]string{"delta": string(argsJSON)})
	delta := sse.Event{
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
	done := sse.Event{
		Type: "response.output_item.done",
		Data: string(doneData),
	}

	state.OutputIndex++
	return append(flushEvents, added, delta, done)
}

func thoughtEvents(part *genai.Part, state *EmitState) []sse.Event {
	if part.Text != "" {
		return emitReasoningDelta(part.Text, state)
	}
	if len(part.ThoughtSignature) > 0 {
		state.ThoughtSig = part.ThoughtSignature
		state.HasThought = true
	}
	return nil
}

func flushThought(state *EmitState) []sse.Event {
	if !state.HasThought || len(state.ThoughtSig) == 0 {
		state.HasThought = false
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
	state.ThoughtSig = nil
	return []sse.Event{{
		Type: "response.output_item.done",
		Data: string(data),
	}}
}

func ExtractGeminiUsage(chunk *genai.GenerateContentResponse) *protocol.UsageResponse {
	if chunk == nil || chunk.UsageMetadata == nil {
		return nil
	}
	m := chunk.UsageMetadata
	// PromptTokenCount includes CachedContentTokenCount (per Gemini SDK docs).
	// CandidatesTokenCount excludes ThoughtsTokenCount — we combine them to match
	// OpenAI's convention where output_tokens includes reasoning_tokens.
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

func ExtractFinishReason(chunk *genai.GenerateContentResponse) genai.FinishReason {
	if chunk == nil || len(chunk.Candidates) == 0 {
		return ""
	}
	return chunk.Candidates[0].FinishReason
}

var finishReasonErrors = map[genai.FinishReason]string{
	genai.FinishReasonMaxTokens:             "output truncated: token limit reached",
	genai.FinishReasonSafety:                "output blocked by safety filter",
	genai.FinishReasonRecitation:            "output blocked by recitation filter",
	genai.FinishReasonMalformedFunctionCall: "malformed function call",
	genai.FinishReasonBlocklist:             "output blocked by blocklist filter",
	genai.FinishReasonProhibitedContent:     "output blocked: prohibited content",
	genai.FinishReasonSPII:                  "output blocked: sensitive personal information detected",
}

func FinishReasonToEvent(reason genai.FinishReason) *sse.Event {
	return sse.FinishReasonToEvent(finishReasonErrors, reason)
}

func ExtractPromptFeedback(chunk *genai.GenerateContentResponse) string {
	if chunk == nil || chunk.PromptFeedback == nil || chunk.PromptFeedback.BlockReason == "" {
		return ""
	}
	return fmt.Sprintf("I'm unable to process this request (blocked: %s).", chunk.PromptFeedback.BlockReason)
}
