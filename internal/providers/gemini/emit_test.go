package gemini

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/genai"
	"hermes-logos/internal/protocol"
)

func TestChunkToEvents(t *testing.T) {
	tests := []struct {
		name   string
		chunk  *genai.GenerateContentResponse
		state  *EmitState
		check  func(t *testing.T, events []SSEEvent, state *EmitState)
	}{
		{
			name:  "nil chunk",
			chunk: nil,
			state: &EmitState{},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
		{
			name: "text part",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{Text: "hello"}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.output_text.delta" {
					t.Errorf("type = %q", events[0].Type)
				}
				assertJSONField(t, events[0].Data, "delta", "hello")
			},
		},
		{
			name: "function call part",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{
							FunctionCall: &genai.FunctionCall{
								Name: "search",
								Args: map[string]any{"q": "test"},
							},
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 3 {
					t.Fatalf("want 3 events, got %d", len(events))
				}
				if events[0].Type != "response.output_item.added" {
					t.Errorf("event[0].type = %q, want response.output_item.added", events[0].Type)
				}
				if events[1].Type != "response.function_call_arguments.delta" {
					t.Errorf("event[1].type = %q, want response.function_call_arguments.delta", events[1].Type)
				}
				if events[2].Type != "response.output_item.done" {
					t.Errorf("event[2].type = %q, want response.output_item.done", events[2].Type)
				}
				if state.NextCallID != 1 {
					t.Errorf("NextCallID = %d, want 1", state.NextCallID)
				}
				if state.OutputIndex != 1 {
					t.Errorf("OutputIndex = %d, want 1", state.OutputIndex)
				}
			},
		},
		{
			name: "function call with thought signature",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{
							FunctionCall: &genai.FunctionCall{
								Name: "search",
								Args: map[string]any{"q": "test"},
							},
							ThoughtSignature: []byte("sig123"),
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 3 {
					t.Fatalf("want 3 events, got %d", len(events))
				}
				doneData := events[2].Data
				assertJSONContains(t, doneData, "extra_content")
			},
		},
		{
			name: "function call without signature has no extra_content",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{
							FunctionCall: &genai.FunctionCall{
								Name: "search",
								Args: map[string]any{},
							},
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 3 {
					t.Fatalf("want 3 events, got %d", len(events))
				}
				doneData := events[2].Data
				assertJSONNotContains(t, doneData, "extra_content")
			},
		},
		{
			name: "thought text part",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{
							Text:    "thinking...",
							Thought: true,
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.reasoning_summary_text.delta" {
					t.Errorf("type = %q", events[0].Type)
				}
				if !state.HasThought {
					t.Error("expected HasThought=true")
				}
			},
		},
		{
			name: "thought signature part",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{
							Thought:          true,
							ThoughtSignature: []byte("sig123"),
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events for signature-only part, got %d", len(events))
				}
				if string(state.ThoughtSig) != "sig123" {
					t.Errorf("ThoughtSig = %q", state.ThoughtSig)
				}
			},
		},
		{
			name: "text after thought flushes reasoning item",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{Text: "output text"}},
					},
				}},
			},
			state: &EmitState{HasThought: true, ThoughtSig: []byte("sig"), ThoughtText: "thought"},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 2 {
					t.Fatalf("want 2 events (flush + text), got %d", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("flush event type = %q", events[0].Type)
				}
				if events[1].Type != "response.output_text.delta" {
					t.Errorf("text event type = %q", events[1].Type)
				}
				if state.HasThought {
					t.Error("expected HasThought=false after flush")
				}
			},
		},
		{
			name: "function call after thought flushes reasoning item",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{
							FunctionCall: &genai.FunctionCall{Name: "fn", Args: map[string]any{}},
						}},
					},
				}},
			},
			state: &EmitState{HasThought: true, ThoughtSig: []byte("sig"), ThoughtText: "thought"},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 4 {
					t.Fatalf("want 4 events (flush + 3 fc), got %d", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("flush event type = %q", events[0].Type)
				}
			},
		},
		{
			name: "no candidates",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []SSEEvent, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := ChunkToEvents(tt.chunk, tt.state)
			tt.check(t, events, tt.state)
		})
	}
}

func TestExtractGeminiUsage(t *testing.T) {
	tests := []struct {
		name  string
		chunk *genai.GenerateContentResponse
		want  *protocol.UsageResponse
	}{
		{
			name:  "nil chunk",
			chunk: nil,
			want:  nil,
		},
		{
			name:  "nil usage metadata",
			chunk: &genai.GenerateContentResponse{},
			want:  nil,
		},
		{
			name: "basic usage",
			chunk: &genai.GenerateContentResponse{
				UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
					PromptTokenCount:     100,
					CandidatesTokenCount: 50,
					TotalTokenCount:      150,
				},
			},
			want: &protocol.UsageResponse{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
			},
		},
		{
			name: "with cached and reasoning tokens",
			chunk: &genai.GenerateContentResponse{
				UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
					PromptTokenCount:        200,
					CandidatesTokenCount:    80,
					TotalTokenCount:         280,
					CachedContentTokenCount: 30,
					ThoughtsTokenCount:      20,
				},
			},
			want: &protocol.UsageResponse{
				InputTokens:         200,
				OutputTokens:        80,
				TotalTokens:         280,
				InputTokensDetails:  &protocol.PromptTokensDetails{CachedTokens: 30},
				OutputTokensDetails: &protocol.OutputTokensDetails{ReasoningTokens: 20},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractGeminiUsage(tt.chunk)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildCompletedEvent(t *testing.T) {
	usage := &protocol.UsageResponse{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}
	event := BuildCompletedEvent(usage)
	if event.Type != "response.completed" {
		t.Errorf("type = %q, want response.completed", event.Type)
	}
	var parsed map[string]json.RawMessage
	json.Unmarshal([]byte(event.Data), &parsed)
	if _, ok := parsed["response"]; !ok {
		t.Error("expected response field in data")
	}
}

func assertJSONField(t *testing.T, data, key, want string) {
	t.Helper()
	var m map[string]string
	if json.Unmarshal([]byte(data), &m) != nil {
		t.Fatalf("invalid json: %s", data)
	}
	if m[key] != want {
		t.Errorf("%s = %q, want %q", key, m[key], want)
	}
}

func assertJSONContains(t *testing.T, data, key string) {
	t.Helper()
	var outer map[string]json.RawMessage
	if json.Unmarshal([]byte(data), &outer) != nil {
		t.Fatalf("invalid json: %s", data)
	}
	var item map[string]json.RawMessage
	if json.Unmarshal(outer["item"], &item) != nil {
		t.Fatal("missing item in data")
	}
	if _, ok := item[key]; !ok {
		t.Errorf("expected %q in item", key)
	}
}

func assertJSONNotContains(t *testing.T, data, key string) {
	t.Helper()
	var outer map[string]json.RawMessage
	if json.Unmarshal([]byte(data), &outer) != nil {
		t.Fatalf("invalid json: %s", data)
	}
	var item map[string]json.RawMessage
	if json.Unmarshal(outer["item"], &item) != nil {
		t.Fatal("missing item in data")
	}
	if _, ok := item[key]; ok {
		t.Errorf("did not expect %q in item", key)
	}
}
