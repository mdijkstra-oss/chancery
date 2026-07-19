package completions

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/matthijn/hermes-logos/internal/protocol"
	"github.com/matthijn/hermes-logos/internal/providers/sse"
)

func TestChunkToEvents(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		state *EmitState
		check func(t *testing.T, events []sse.Event, state *EmitState)
	}{
		{
			name:  "text delta",
			data:  `{"choices":[{"delta":{"content":"hello"},"finish_reason":null}]}`,
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
			name:  "empty content delta ignored",
			data:  `{"choices":[{"delta":{"content":""},"finish_reason":null}]}`,
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
		{
			name:  "null content delta ignored",
			data:  `{"choices":[{"delta":{},"finish_reason":null}]}`,
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
		{
			name:  "tool call added",
			data:  `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"search","arguments":""}}]},"finish_reason":null}]}`,
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.output_item.added" {
					t.Errorf("type = %q", events[0].Type)
				}
				if state.ActiveCalls[0] == nil {
					t.Fatal("expected active call at index 0")
				}
				if state.ActiveCalls[0].Name != "search" {
					t.Errorf("name = %q", state.ActiveCalls[0].Name)
				}
			},
		},
		{
			name: "tool call arguments delta",
			data: `{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"q\":"}}]},"finish_reason":null}]}`,
			state: &EmitState{ActiveCalls: map[int]*activeCall{
				0: {ID: "call_abc", Name: "search", Arguments: ""},
			}},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.function_call_arguments.delta" {
					t.Errorf("type = %q", events[0].Type)
				}
				if state.ActiveCalls[0].Arguments != `{"q":` {
					t.Errorf("accumulated = %q", state.ActiveCalls[0].Arguments)
				}
			},
		},
		{
			name: "finish stop flushes active calls",
			data: `{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			state: &EmitState{ActiveCalls: map[int]*activeCall{
				0: {ID: "call_abc", Name: "search", Arguments: `{"q":"test"}`},
			}},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("type = %q", events[0].Type)
				}
				if state.ActiveCalls != nil {
					t.Error("expected ActiveCalls to be cleared")
				}
			},
		},
		{
			name: "finish tool_calls flushes multiple calls in order",
			data: `{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
			state: &EmitState{ActiveCalls: map[int]*activeCall{
				0: {ID: "c1", Name: "search", Arguments: "{}"},
				1: {ID: "c2", Name: "read", Arguments: `{"f":"a"}`},
			}},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 2 {
					t.Fatalf("want 2 events, got %d", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("event[0].type = %q", events[0].Type)
				}
				if events[1].Type != "response.output_item.done" {
					t.Errorf("event[1].type = %q", events[1].Type)
				}
				if state.OutputIndex != 2 {
					t.Errorf("OutputIndex = %d, want 2", state.OutputIndex)
				}
			},
		},
		{
			name:  "mixed content and tool call in same chunk",
			data:  `{"choices":[{"delta":{"content":"thinking","tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"fn","arguments":""}}]},"finish_reason":null}]}`,
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 2 {
					t.Fatalf("want 2 events, got %d", len(events))
				}
				if events[0].Type != "response.output_text.delta" {
					t.Errorf("event[0].type = %q", events[0].Type)
				}
				if events[1].Type != "response.output_item.added" {
					t.Errorf("event[1].type = %q", events[1].Type)
				}
			},
		},
		{
			name:  "reasoning_content delta",
			data:  `{"choices":[{"delta":{"reasoning_content":"let me think"},"finish_reason":null}]}`,
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.reasoning_summary_text.delta" {
					t.Errorf("type = %q", events[0].Type)
				}
				assertJSONField(t, events[0].Data, "delta", "let me think")
				if state.ReasoningText != "let me think" {
					t.Errorf("ReasoningText = %q", state.ReasoningText)
				}
			},
		},
		{
			name:  "reasoning flushed when content starts",
			data:  `{"choices":[{"delta":{"content":"answer"},"finish_reason":null}]}`,
			state: &EmitState{ReasoningText: "I was thinking"},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 2 {
					t.Fatalf("want 2 events, got %d", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("event[0].type = %q", events[0].Type)
				}
				if events[1].Type != "response.output_text.delta" {
					t.Errorf("event[1].type = %q", events[1].Type)
				}
				if state.ReasoningText != "" {
					t.Errorf("ReasoningText should be cleared, got %q", state.ReasoningText)
				}
				if state.OutputIndex != 1 {
					t.Errorf("OutputIndex = %d, want 1", state.OutputIndex)
				}
			},
		},
		{
			name:  "reasoning flushed when tool call starts",
			data:  `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"search","arguments":""}}]},"finish_reason":null}]}`,
			state: &EmitState{ReasoningText: "thinking about tools"},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 2 {
					t.Fatalf("want 2 events, got %d", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("event[0].type = %q", events[0].Type)
				}
				if events[1].Type != "response.output_item.added" {
					t.Errorf("event[1].type = %q", events[1].Type)
				}
				if state.ReasoningText != "" {
					t.Errorf("ReasoningText should be cleared")
				}
			},
		},
		{
			name:  "reasoning flushed on finish with no content",
			data:  `{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			state: &EmitState{ReasoningText: "orphaned reasoning"},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("type = %q", events[0].Type)
				}
				if state.ReasoningText != "" {
					t.Errorf("ReasoningText should be cleared")
				}
			},
		},
		{
			name:  "empty reasoning_content ignored",
			data:  `{"choices":[{"delta":{"reasoning_content":""},"finish_reason":null}]}`,
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
		{
			name:  "invalid JSON returns nil",
			data:  `not json`,
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
		{
			name:  "no choices returns nil",
			data:  `{"choices":[]}`,
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := ChunkToEvents([]byte(tt.data), tt.state)
			tt.check(t, events, tt.state)
		})
	}
}

func TestFlushActiveCalls(t *testing.T) {
	tests := []struct {
		name  string
		state *EmitState
		check func(t *testing.T, events []sse.Event, state *EmitState)
	}{
		{
			name:  "no active calls",
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
		{
			name: "single call",
			state: &EmitState{ActiveCalls: map[int]*activeCall{
				0: {ID: "c1", Name: "search", Arguments: `{"q":"test"}`},
			}},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("type = %q", events[0].Type)
				}
				if state.OutputIndex != 1 {
					t.Errorf("OutputIndex = %d, want 1", state.OutputIndex)
				}
				if state.ActiveCalls != nil {
					t.Error("expected ActiveCalls cleared")
				}
			},
		},
		{
			name: "multiple calls ordered by index",
			state: &EmitState{ActiveCalls: map[int]*activeCall{
				1: {ID: "c2", Name: "read", Arguments: "{}"},
				0: {ID: "c1", Name: "search", Arguments: "{}"},
			}},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 2 {
					t.Fatalf("want 2 events, got %d", len(events))
				}
				assertItemCallID(t, events[0].Data, "c1")
				assertItemCallID(t, events[1].Data, "c2")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := FlushActiveCalls(tt.state)
			tt.check(t, events, tt.state)
		})
	}
}

func TestFlushReasoning(t *testing.T) {
	tests := []struct {
		name  string
		state *EmitState
		check func(t *testing.T, events []sse.Event, state *EmitState)
	}{
		{
			name:  "no reasoning text",
			state: &EmitState{},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("want 0 events, got %d", len(events))
				}
			},
		},
		{
			name:  "flushes reasoning item",
			state: &EmitState{ReasoningText: "deep thought"},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("want 1 event, got %d", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("type = %q", events[0].Type)
				}
				var parsed struct {
					Item struct {
						Type         string `json:"type"`
						ID           string `json:"id"`
						ExtraContent struct {
							DeepSeek struct {
								ReasoningContent string `json:"reasoning_content"`
							} `json:"deepseek"`
						} `json:"extra_content"`
					} `json:"item"`
				}
				if json.Unmarshal([]byte(events[0].Data), &parsed) != nil {
					t.Fatalf("invalid json: %s", events[0].Data)
				}
				if parsed.Item.Type != "reasoning" {
					t.Errorf("item.type = %q", parsed.Item.Type)
				}
				if parsed.Item.ExtraContent.DeepSeek.ReasoningContent != "deep thought" {
					t.Errorf("reasoning_content = %q", parsed.Item.ExtraContent.DeepSeek.ReasoningContent)
				}
				if state.ReasoningText != "" {
					t.Error("ReasoningText should be cleared")
				}
				if state.OutputIndex != 1 {
					t.Errorf("OutputIndex = %d, want 1", state.OutputIndex)
				}
			},
		},
		{
			name:  "idempotent after flush",
			state: &EmitState{ReasoningText: "thought"},
			check: func(t *testing.T, events []sse.Event, state *EmitState) {
				second := FlushReasoning(state)
				if len(second) != 0 {
					t.Errorf("second flush should return 0 events, got %d", len(second))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := FlushReasoning(tt.state)
			tt.check(t, events, tt.state)
		})
	}
}

func TestExtractUsage(t *testing.T) {
	tests := []struct {
		name string
		data string
		want *protocol.UsageResponse
	}{
		{
			name: "nil when no usage",
			data: `{"choices":[{"delta":{"content":"hi"}}]}`,
			want: nil,
		},
		{
			name: "basic usage",
			data: `{"choices":[],"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}`,
			want: &protocol.UsageResponse{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
			},
		},
		{
			name: "with cached and reasoning tokens",
			data: `{"choices":[],"usage":{"prompt_tokens":200,"completion_tokens":80,"total_tokens":280,"prompt_tokens_details":{"cached_tokens":30},"completion_tokens_details":{"reasoning_tokens":20}}}`,
			want: &protocol.UsageResponse{
				InputTokens:         200,
				OutputTokens:        80,
				TotalTokens:         280,
				InputTokensDetails:  &protocol.PromptTokensDetails{CachedTokens: 30},
				OutputTokensDetails: &protocol.OutputTokensDetails{ReasoningTokens: 20},
			},
		},
		{
			name: "invalid JSON",
			data: `not json`,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractUsage([]byte(tt.data))
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
		t.Errorf("type = %q", event.Type)
	}
	var parsed map[string]json.RawMessage
	if json.Unmarshal([]byte(event.Data), &parsed) != nil {
		t.Fatalf("invalid json: %s", event.Data)
	}
	if _, ok := parsed["response"]; !ok {
		t.Error("expected response field")
	}
}

func TestExtractFinishReason(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "content_filter",
			data: `{"choices":[{"delta":{},"finish_reason":"content_filter"}]}`,
			want: "content_filter",
		},
		{
			name: "length",
			data: `{"choices":[{"delta":{},"finish_reason":"length"}]}`,
			want: "length",
		},
		{
			name: "stop",
			data: `{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			want: "stop",
		},
		{
			name: "null finish_reason",
			data: `{"choices":[{"delta":{"content":"hi"},"finish_reason":null}]}`,
			want: "",
		},
		{
			name: "no choices",
			data: `{"choices":[]}`,
			want: "",
		},
		{
			name: "invalid JSON",
			data: `not json`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFinishReason([]byte(tt.data))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFinishReasonToEvent(t *testing.T) {
	tests := []struct {
		name       string
		reason     string
		wantNil    bool
		wantType   string
		wantErrMsg string
	}{
		{
			name:       "content_filter returns failed event",
			reason:     "content_filter",
			wantType:   "content_filter",
			wantErrMsg: "output blocked by content filter",
		},
		{
			name:       "length returns failed event",
			reason:     "length",
			wantType:   "length",
			wantErrMsg: "output truncated: token limit reached",
		},
		{
			name:    "stop returns nil",
			reason:  "stop",
			wantNil: true,
		},
		{
			name:    "unknown reason returns nil",
			reason:  "tool_calls",
			wantNil: true,
		},
		{
			name:    "empty reason returns nil",
			reason:  "",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := FinishReasonToEvent(tt.reason)
			if tt.wantNil {
				if event != nil {
					t.Fatalf("expected nil, got %+v", event)
				}
				return
			}
			if event == nil {
				t.Fatal("expected non-nil event")
			}
			if event.Type != "response.failed" {
				t.Errorf("event type = %q", event.Type)
			}
			var parsed struct {
				Response struct {
					Error struct {
						Type    string `json:"type"`
						Message string `json:"message"`
					} `json:"error"`
				} `json:"response"`
			}
			if json.Unmarshal([]byte(event.Data), &parsed) != nil {
				t.Fatalf("invalid json: %s", event.Data)
			}
			if parsed.Response.Error.Type != tt.wantType {
				t.Errorf("error.type = %q, want %q", parsed.Response.Error.Type, tt.wantType)
			}
			if parsed.Response.Error.Message != tt.wantErrMsg {
				t.Errorf("error.message = %q, want %q", parsed.Response.Error.Message, tt.wantErrMsg)
			}
		})
	}
}

func TestBuildFailedEvent(t *testing.T) {
	event := sse.BuildFailedEvent("rate_limit", "too many requests")
	if event.Type != "response.failed" {
		t.Errorf("type = %q", event.Type)
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
		t.Errorf("status = %q", parsed.Response.Status)
	}
	if parsed.Response.Error.Type != "rate_limit" {
		t.Errorf("error.type = %q", parsed.Response.Error.Type)
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

func assertItemCallID(t *testing.T, data, wantID string) {
	t.Helper()
	var outer map[string]json.RawMessage
	if json.Unmarshal([]byte(data), &outer) != nil {
		t.Fatalf("invalid json: %s", data)
	}
	var item map[string]string
	if json.Unmarshal(outer["item"], &item) != nil {
		t.Fatal("missing item")
	}
	if item["call_id"] != wantID {
		t.Errorf("call_id = %q, want %q", item["call_id"], wantID)
	}
}
