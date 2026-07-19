package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/matthijn/hermes-logos/internal/providers/sse"
)

func TestHandleEvent(t *testing.T) {
	cases := []struct {
		name      string
		eventType string
		data      string
		state     *EmitState
		check     func(t *testing.T, events []sse.Event, state *EmitState)
	}{
		{
			"message_start extracts usage",
			"message_start",
			`{"message":{"usage":{"input_tokens":100,"cache_read_input_tokens":50,"cache_creation_input_tokens":25}}}`,
			&EmitState{},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("events = %d, want 0", len(events))
				}
				if state.InputTokens != 100 {
					t.Errorf("InputTokens = %d, want 100", state.InputTokens)
				}
				if state.CacheRead != 50 {
					t.Errorf("CacheRead = %d, want 50", state.CacheRead)
				}
				if state.CacheCreation != 25 {
					t.Errorf("CacheCreation = %d, want 25", state.CacheCreation)
				}
			},
		},
		{
			"text content block start",
			"content_block_start",
			`{"index":0,"content_block":{"type":"text","text":""}}`,
			&EmitState{},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("events = %d, want 0", len(events))
				}
				if state.ActiveBlockType != "text" {
					t.Errorf("ActiveBlockType = %q, want text", state.ActiveBlockType)
				}
			},
		},
		{
			"tool_use content block start",
			"content_block_start",
			`{"index":1,"content_block":{"type":"tool_use","id":"call_123","name":"search"}}`,
			&EmitState{},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("events = %d, want 1", len(events))
				}
				if events[0].Type != "response.output_item.added" {
					t.Errorf("type = %q, want response.output_item.added", events[0].Type)
				}
				if state.ToolCallID != "call_123" {
					t.Errorf("ToolCallID = %q, want call_123", state.ToolCallID)
				}
			},
		},
		{
			"thinking content block start",
			"content_block_start",
			`{"index":0,"content_block":{"type":"thinking","thinking":"","signature":""}}`,
			&EmitState{},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("events = %d, want 0", len(events))
				}
				if state.ActiveBlockType != "thinking" {
					t.Errorf("ActiveBlockType = %q, want thinking", state.ActiveBlockType)
				}
			},
		},
		{
			"text delta",
			"content_block_delta",
			`{"index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			&EmitState{ActiveBlockType: "text"},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("events = %d, want 1", len(events))
				}
				if events[0].Type != "response.output_text.delta" {
					t.Errorf("type = %q", events[0].Type)
				}
			},
		},
		{
			"input_json delta",
			"content_block_delta",
			`{"index":1,"delta":{"type":"input_json_delta","partial_json":"{\"q\":"}}`,
			&EmitState{ActiveBlockType: "tool_use", ToolCallID: "c1"},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("events = %d, want 1", len(events))
				}
				if events[0].Type != "response.function_call_arguments.delta" {
					t.Errorf("type = %q", events[0].Type)
				}
				if state.ToolCallJSON != `{"q":` {
					t.Errorf("ToolCallJSON = %q", state.ToolCallJSON)
				}
			},
		},
		{
			"thinking delta",
			"content_block_delta",
			`{"index":0,"delta":{"type":"thinking_delta","thinking":"Let me think..."}}`,
			&EmitState{ActiveBlockType: "thinking"},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("events = %d, want 1", len(events))
				}
				if events[0].Type != "response.reasoning_summary_text.delta" {
					t.Errorf("type = %q", events[0].Type)
				}
				if state.ThinkingText != "Let me think..." {
					t.Errorf("ThinkingText = %q", state.ThinkingText)
				}
			},
		},
		{
			"signature delta accumulates",
			"content_block_delta",
			`{"index":0,"delta":{"type":"signature_delta","signature":"abc123"}}`,
			&EmitState{ActiveBlockType: "thinking"},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("events = %d, want 0", len(events))
				}
				if state.ThinkingSig != "abc123" {
					t.Errorf("ThinkingSig = %q, want abc123", state.ThinkingSig)
				}
			},
		},
		{
			"content block stop for thinking emits done",
			"content_block_stop",
			`{"index":0}`,
			&EmitState{ActiveBlockType: "thinking", ThinkingText: "I thought", ThinkingSig: "sig123", OutputIndex: 0},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("events = %d, want 1", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("type = %q, want response.output_item.done", events[0].Type)
				}
				if state.OutputIndex != 1 {
					t.Errorf("OutputIndex = %d, want 1", state.OutputIndex)
				}
				var parsed struct {
					Item struct {
						Type string `json:"type"`
					} `json:"item"`
				}
				json.Unmarshal([]byte(events[0].Data), &parsed)
				if parsed.Item.Type != "reasoning" {
					t.Errorf("item.type = %q, want reasoning", parsed.Item.Type)
				}
			},
		},
		{
			"content block stop for tool_use emits done",
			"content_block_stop",
			`{"index":1}`,
			&EmitState{ActiveBlockType: "tool_use", ToolCallID: "c1", ToolCallName: "search", ToolCallJSON: `{"q":"test"}`, OutputIndex: 1},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("events = %d, want 1", len(events))
				}
				if events[0].Type != "response.output_item.done" {
					t.Errorf("type = %q", events[0].Type)
				}
				if state.OutputIndex != 2 {
					t.Errorf("OutputIndex = %d, want 2", state.OutputIndex)
				}
			},
		},
		{
			"message delta with max_tokens emits failed",
			"message_delta",
			`{"delta":{"stop_reason":"max_tokens"},"usage":{"output_tokens":100}}`,
			&EmitState{},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("events = %d, want 1", len(events))
				}
				if events[0].Type != "response.failed" {
					t.Errorf("type = %q, want response.failed", events[0].Type)
				}
			},
		},
		{
			"message delta with end_turn emits nothing",
			"message_delta",
			`{"delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":50}}`,
			&EmitState{},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("events = %d, want 0", len(events))
				}
			},
		},
		{
			"error event",
			"error",
			`{"error":{"type":"overloaded_error","message":"Overloaded"}}`,
			&EmitState{},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 1 {
					t.Fatalf("events = %d, want 1", len(events))
				}
				if events[0].Type != "response.failed" {
					t.Errorf("type = %q, want response.failed", events[0].Type)
				}
			},
		},
		{
			"ping is ignored",
			"ping",
			`{"type":"ping"}`,
			&EmitState{},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("events = %d, want 0", len(events))
				}
			},
		},
		{
			"message_stop is ignored",
			"message_stop",
			`{"type":"message_stop"}`,
			&EmitState{},
			func(t *testing.T, events []sse.Event, state *EmitState) {
				if len(events) != 0 {
					t.Errorf("events = %d, want 0", len(events))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := HandleEvent(tc.eventType, []byte(tc.data), tc.state)
			tc.check(t, events, tc.state)
		})
	}
}

func TestExtractUsage(t *testing.T) {
	state := &EmitState{
		InputTokens:   100,
		CacheRead:     50,
		CacheCreation: 25,
	}
	data := []byte(`{"delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":200}}`)

	usage := ExtractUsage(state, data)
	if usage.InputTokens != 175 {
		t.Errorf("InputTokens = %d, want 175 (100+50+25)", usage.InputTokens)
	}
	if usage.OutputTokens != 200 {
		t.Errorf("OutputTokens = %d, want 200", usage.OutputTokens)
	}
	if usage.TotalTokens != 375 {
		t.Errorf("TotalTokens = %d, want 375", usage.TotalTokens)
	}
	if usage.InputTokensDetails == nil {
		t.Fatal("InputTokensDetails is nil")
	}
	if usage.InputTokensDetails.CachedTokens != 50 {
		t.Errorf("CachedTokens = %d, want 50", usage.InputTokensDetails.CachedTokens)
	}
	if usage.InputTokensDetails.CacheCreationTokens != 25 {
		t.Errorf("CacheCreationTokens = %d, want 25", usage.InputTokensDetails.CacheCreationTokens)
	}
}
