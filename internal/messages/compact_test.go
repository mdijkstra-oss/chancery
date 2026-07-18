package messages

import (
	"encoding/json"
	"testing"
)

func TestStripForCompaction(t *testing.T) {
	cases := []struct {
		name     string
		messages []json.RawMessage
		expected int
	}{
		{"no messages", nil, 0},
		{"drops last from user and assistant", []json.RawMessage{
			json.RawMessage(`{"role":"user","content":"hi"}`),
			json.RawMessage(`{"role":"assistant","content":"hello"}`),
		}, 1},
		{"strips system messages then drops last", []json.RawMessage{
			json.RawMessage(`{"role":"system","content":"foo"}`),
			json.RawMessage(`{"role":"user","content":"hi"}`),
		}, 0},
		{"strips reasoning and drops last", []json.RawMessage{
			json.RawMessage(`{"type":"reasoning","id":"r1","summary":[{"type":"summary_text","text":"thinking"}]}`),
			json.RawMessage(`{"role":"user","content":"hi"}`),
			json.RawMessage(`{"role":"assistant","content":"hello"}`),
		}, 1},
		{"keeps function_call drops last", []json.RawMessage{
			json.RawMessage(`{"type":"function_call","call_id":"c1","name":"shell","arguments":"{}"}`),
			json.RawMessage(`{"type":"function_call_output","call_id":"c1","output":"ok"}`),
			json.RawMessage(`{"role":"user","content":"hi"}`),
		}, 2},
		{"mixed strips system and reasoning drops last", []json.RawMessage{
			json.RawMessage(`{"role":"system","content":"<!-- prompt: planning -->"}`),
			json.RawMessage(`{"role":"user","content":"hi"}`),
			json.RawMessage(`{"type":"reasoning","id":"r1"}`),
			json.RawMessage(`{"type":"function_call","call_id":"c1","name":"shell","arguments":"{}"}`),
			json.RawMessage(`{"role":"assistant","content":"hello"}`),
		}, 2},
		{"invalid json stripped", []json.RawMessage{
			json.RawMessage(`not json`),
			json.RawMessage(`{"role":"user","content":"hi"}`),
		}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StripForCompaction(tc.messages)
			if len(got) != tc.expected {
				t.Errorf("StripForCompaction() returned %d messages, want %d", len(got), tc.expected)
			}
		})
	}
}

func TestShouldCompact(t *testing.T) {
	cases := []struct {
		name      string
		compactAt int
		tokens    int
		expected  bool
	}{
		{"compactAt zero", 0, 100000, false},
		{"tokens below threshold", 500000, 400000, false},
		{"tokens above threshold", 500000, 600000, true},
		{"tokens at threshold", 500000, 500000, false},
		{"tokens just above", 500000, 500001, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldCompact(tc.compactAt, tc.tokens)
			if got != tc.expected {
				t.Errorf("ShouldCompact(%d, %d) = %v, want %v", tc.compactAt, tc.tokens, got, tc.expected)
			}
		})
	}
}
