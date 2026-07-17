package gemini

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/genai"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/providers/sse"
)

func TestChunkToEvents(t *testing.T) {
	tests := []struct {
		name  string
		chunk *genai.GenerateContentResponse
		state *EmitState
		check func(t *testing.T, events []sse.Event, state *EmitState)
	}{
		{
			name:  "nil chunk",
			chunk: nil,
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
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
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
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
								ID:   "fc_abc123",
								Name: "search",
								Args: map[string]any{"q": "test"},
							},
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
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
				assertJSONItemField(t, events[0].Data, "call_id", "fc_abc123")
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
								ID:   "fc_sig1",
								Name: "search",
								Args: map[string]any{"q": "test"},
							},
							ThoughtSignature: []byte("sig123"),
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
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
								ID:   "fc_nosig",
								Name: "search",
								Args: map[string]any{},
							},
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 3 {
					t.Fatalf("want 3 events, got %d", len(events))
				}
				doneData := events[2].Data
				assertJSONNotContains(t, doneData, "extra_content")
			},
		},
		{
			name: "text part with thought signature stores sig",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{
							Text:             "final answer",
							ThoughtSignature: []byte("textsig"),
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.output_text.delta" {
					t.Errorf("type = %q, want response.output_text.delta", events[0].Type)
				}
				if string(state.ThoughtSig) != "textsig" {
					t.Errorf("ThoughtSig = %q, want textsig", string(state.ThoughtSig))
				}
				if !state.HasThought {
					t.Error("expected HasThought=true")
				}
			},
		},
		{
			name: "standalone thought signature without thought flag",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{
							ThoughtSignature: []byte("standalone"),
						}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
				if string(state.ThoughtSig) != "standalone" {
					t.Errorf("ThoughtSig = %q, want standalone", string(state.ThoughtSig))
				}
			},
		},
		{
			name: "empty part is silently skipped",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events for empty part, got %d", len(events))
				}
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
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
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
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
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
			state: &EmitState{HasThought: true, ThoughtSig: []byte("sig")},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
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
			name: "function call without sig inherits thought sig",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{
							FunctionCall: &genai.FunctionCall{ID: "fc_flush", Name: "fn", Args: map[string]any{}},
						}},
					},
				}},
			},
			state: &EmitState{HasThought: true, ThoughtSig: []byte("sig")},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 3 {
					t.Fatalf("want 3 events (3 fc), got %d", len(events))
				}
				if events[0].Type != "response.output_item.added" {
					t.Errorf("event[0] type = %q, want response.output_item.added", events[0].Type)
				}
				assertJSONContains(t, events[2].Data, "extra_content")
			},
		},
		{
			name: "no candidates",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
		{
			name: "leaked think reroutes to reasoning",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{Text: "before<think>thinking</think>after"}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 3 {
					t.Fatalf("want 3 events, got %d", len(events))
				}
				if events[0].Type != "response.output_text.delta" {
					t.Errorf("event[0].type = %q", events[0].Type)
				}
				assertJSONField(t, events[0].Data, "delta", "before")
				if events[1].Type != "response.reasoning_summary_text.delta" {
					t.Errorf("event[1].type = %q", events[1].Type)
				}
				assertJSONField(t, events[1].Data, "delta", "thinking")
				if events[2].Type != "response.output_text.delta" {
					t.Errorf("event[2].type = %q", events[2].Type)
				}
				assertJSONField(t, events[2].Data, "delta", "after")
			},
		},
		{
			name: "entirely leaked think",
			chunk: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					Content: &genai.Content{
						Parts: []*genai.Part{{Text: "<think>all reasoning</think>"}},
					},
				}},
			},
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.reasoning_summary_text.delta" {
					t.Errorf("type = %q", events[0].Type)
				}
				assertJSONField(t, events[0].Data, "delta", "all reasoning")
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

func TestFilterLeakedThinking(t *testing.T) {
	tests := []struct {
		name            string
		text            string
		state           *EmitState
		want            []textSegment
		wantSuppressing bool
		wantLeakedBuf   string
	}{
		{
			name:  "no tags",
			text:  "hello world",
			state: &EmitState{},
			want:  []textSegment{{"hello world", false}},
		},
		{
			name:  "complete think block",
			text:  "<think>reasoning here</think>",
			state: &EmitState{},
			want:  []textSegment{{"reasoning here", true}},
		},
		{
			name:  "complete thought block",
			text:  "<thought>reasoning</thought>output",
			state: &EmitState{},
			want:  []textSegment{{"reasoning", true}, {"output", false}},
		},
		{
			name:  "complete thinking block",
			text:  "<thinking>deep</thinking>result",
			state: &EmitState{},
			want:  []textSegment{{"deep", true}, {"result", false}},
		},
		{
			name:  "text before and after",
			text:  "before<think>inside</think>after",
			state: &EmitState{},
			want:  []textSegment{{"before", false}, {"inside", true}, {"after", false}},
		},
		{
			name:            "unclosed tag spans chunks",
			text:            "text<think>reasoning",
			state:           &EmitState{},
			want:            []textSegment{{"text", false}, {"reasoning", true}},
			wantSuppressing: true,
		},
		{
			name:  "already suppressing finds close",
			text:  "continued</think>output",
			state: &EmitState{Suppressing: true},
			want:  []textSegment{{"continued", true}, {"output", false}},
		},
		{
			name:          "partial opening tag at end",
			text:          "hello<thin",
			state:         &EmitState{},
			want:          []textSegment{{"hello", false}},
			wantLeakedBuf: "<thin",
		},
		{
			name:            "partial closing tag at end",
			text:            "reasoning</thi",
			state:           &EmitState{Suppressing: true},
			want:            []textSegment{{"reasoning", true}},
			wantSuppressing: true,
			wantLeakedBuf:   "</thi",
		},
		{
			name:            "buffer confirms opening tag",
			text:            "nk>inside",
			state:           &EmitState{LeakedBuf: "<thi"},
			want:            []textSegment{{"inside", true}},
			wantSuppressing: true,
		},
		{
			name:  "buffer rejects tag",
			text:  "gs are cool",
			state: &EmitState{LeakedBuf: "<thin"},
			want:  []textSegment{{"<things are cool", false}},
		},
		{
			name:  "empty text",
			text:  "",
			state: &EmitState{},
			want:  nil,
		},
		{
			name:  "empty tag",
			text:  "<think></think>",
			state: &EmitState{},
			want:  nil,
		},
		{
			name:  "thinking preferred over think at same position",
			text:  "<thinking>deep</thinking>out",
			state: &EmitState{},
			want:  []textSegment{{"deep", true}, {"out", false}},
		},
		{
			name:          "entire input is partial tag",
			text:          "<th",
			state:         &EmitState{},
			want:          nil,
			wantLeakedBuf: "<th",
		},
		{
			name:  "unrelated xml passes through",
			text:  "<div>content</div>",
			state: &EmitState{},
			want:  []textSegment{{"<div>content</div>", false}},
		},
		{
			name:  "multiple blocks",
			text:  "a<think>b</think>c<thought>d</thought>e",
			state: &EmitState{},
			want:  []textSegment{{"a", false}, {"b", true}, {"c", false}, {"d", true}, {"e", false}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterLeakedThinking(tt.text, tt.state)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(textSegment{})); diff != "" {
				t.Errorf("segments (-want +got):\n%s", diff)
			}
			if tt.state.Suppressing != tt.wantSuppressing {
				t.Errorf("Suppressing = %v, want %v", tt.state.Suppressing, tt.wantSuppressing)
			}
			if tt.state.LeakedBuf != tt.wantLeakedBuf {
				t.Errorf("LeakedBuf = %q, want %q", tt.state.LeakedBuf, tt.wantLeakedBuf)
			}
		})
	}
}

func TestIndexOfAny(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		needles []string
		wantIdx int
		wantLen int
	}{
		{"no match", "hello", []string{"<think>"}, -1, 0},
		{"single match", "a<think>b", []string{"<think>"}, 1, 7},
		{"first of two", "a<think>b<thought>c", []string{"<think>", "<thought>"}, 1, 7},
		{"longer wins on tie", "<thinking>x", []string{"<think>", "<thinking>"}, 0, 10},
		{"earlier wins over longer", "a<think>b<thinking>c", []string{"<thinking>", "<think>"}, 1, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, length := indexOfAny(tt.s, tt.needles)
			if idx != tt.wantIdx || length != tt.wantLen {
				t.Errorf("got (%d, %d), want (%d, %d)", idx, length, tt.wantIdx, tt.wantLen)
			}
		})
	}
}

func TestLongestPartialSuffix(t *testing.T) {
	tests := []struct {
		name string
		s    string
		tags []string
		want int
	}{
		{"no match", "hello", []string{"<think>"}, 0},
		{"single char", "x<", []string{"<think>"}, 1},
		{"multi char", "x<thin", []string{"<think>"}, 5},
		{"full tag not partial", "x<think>", []string{"<think>"}, 0},
		{"best across tags", "x<thou", []string{"<think>", "<thought>"}, 5},
		{"entire string is partial", "<th", []string{"<think>", "<thought>"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := longestPartialSuffix(tt.s, tt.tags)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
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
					TotalTokenCount:         300,
					CachedContentTokenCount: 30,
					ThoughtsTokenCount:      20,
				},
			},
			want: &protocol.UsageResponse{
				InputTokens:         200,
				OutputTokens:        100,
				TotalTokens:         300,
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
	event := sse.BuildCompletedEvent(usage)
	if event.Type != "response.completed" {
		t.Errorf("type = %q, want response.completed", event.Type)
	}
	var parsed map[string]json.RawMessage
	json.Unmarshal([]byte(event.Data), &parsed)
	if _, ok := parsed["response"]; !ok {
		t.Error("expected response field in data")
	}
}

func TestBuildFailedEvent(t *testing.T) {
	event := sse.BuildFailedEvent("SAFETY", "output blocked by safety filter")
	if event.Type != "response.failed" {
		t.Errorf("type = %q, want response.failed", event.Type)
	}
	var parsed struct {
		Response struct {
			Status string `json:"status"`
			Error  struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"response"`
	}
	if json.Unmarshal([]byte(event.Data), &parsed) != nil {
		t.Fatalf("invalid json: %s", event.Data)
	}
	if parsed.Response.Status != "failed" {
		t.Errorf("status = %q, want failed", parsed.Response.Status)
	}
	if parsed.Response.Error.Type != "SAFETY" {
		t.Errorf("error.type = %q, want SAFETY", parsed.Response.Error.Type)
	}
	if parsed.Response.Error.Message != "output blocked by safety filter" {
		t.Errorf("error.message = %q", parsed.Response.Error.Message)
	}
}

func TestFinishReasonToEvent(t *testing.T) {
	tests := []struct {
		reason  genai.FinishReason
		wantNil bool
	}{
		{genai.FinishReasonStop, true},
		{"", true},
		{genai.FinishReasonMaxTokens, false},
		{genai.FinishReasonSafety, false},
		{genai.FinishReasonMalformedFunctionCall, false},
		{genai.FinishReasonRecitation, false},
		{genai.FinishReasonBlocklist, false},
		{genai.FinishReasonProhibitedContent, false},
		{genai.FinishReasonSPII, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			event := FinishReasonToEvent(tt.reason)
			if tt.wantNil && event != nil {
				t.Errorf("expected nil for %q, got event", tt.reason)
			}
			if !tt.wantNil && event == nil {
				t.Errorf("expected event for %q, got nil", tt.reason)
			}
			if event != nil && event.Type != "response.failed" {
				t.Errorf("type = %q, want response.failed", event.Type)
			}
		})
	}
}

func TestExtractFinishReason(t *testing.T) {
	tests := []struct {
		name  string
		chunk *genai.GenerateContentResponse
		want  genai.FinishReason
	}{
		{"nil chunk", nil, ""},
		{"no candidates", &genai.GenerateContentResponse{}, ""},
		{"stop", &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonStop}},
		}, genai.FinishReasonStop},
		{"safety", &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{FinishReason: genai.FinishReasonSafety}},
		}, genai.FinishReasonSafety},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFinishReason(tt.chunk)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractPromptFeedback(t *testing.T) {
	tests := []struct {
		name  string
		chunk *genai.GenerateContentResponse
		want  string
	}{
		{"nil chunk", nil, ""},
		{"no feedback", &genai.GenerateContentResponse{}, ""},
		{"safety block", &genai.GenerateContentResponse{
			PromptFeedback: &genai.GenerateContentResponsePromptFeedback{
				BlockReason: genai.BlockedReasonSafety,
			},
		}, "I'm unable to process this request (blocked: SAFETY)."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPromptFeedback(tt.chunk)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
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

func assertJSONItemField(t *testing.T, data, key, want string) {
	t.Helper()
	var outer map[string]json.RawMessage
	if json.Unmarshal([]byte(data), &outer) != nil {
		t.Fatalf("invalid json: %s", data)
	}
	var item map[string]string
	if json.Unmarshal(outer["item"], &item) != nil {
		t.Fatal("missing item in data")
	}
	if item[key] != want {
		t.Errorf("item.%s = %q, want %q", key, item[key], want)
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
