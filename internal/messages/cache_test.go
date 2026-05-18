package messages

import (
	"encoding/json"
	"testing"
)

func TestExtractCacheBreakpoints(t *testing.T) {
	cases := []struct {
		name            string
		messages        []json.RawMessage
		wantLen         int
		wantBreakpoints map[int]bool
	}{
		{
			"no markers",
			[]json.RawMessage{
				json.RawMessage(`{"type":"message","role":"user","content":"hello"}`),
				json.RawMessage(`{"type":"message","role":"assistant","content":"hi"}`),
			},
			2,
			nil,
		},
		{
			"single marker between messages",
			[]json.RawMessage{
				json.RawMessage(`{"type":"message","role":"system","content":"you are helpful"}`),
				json.RawMessage(`{"type":"message","role":"system","content":"<!-- cache -->"}`),
				json.RawMessage(`{"type":"message","role":"user","content":"hello"}`),
			},
			2,
			map[int]bool{0: true},
		},
		{
			"marker at end",
			[]json.RawMessage{
				json.RawMessage(`{"type":"message","role":"user","content":"hello"}`),
				json.RawMessage(`{"type":"message","role":"assistant","content":"hi"}`),
				json.RawMessage(`{"type":"message","role":"system","content":"<!-- cache -->"}`),
			},
			2,
			map[int]bool{1: true},
		},
		{
			"multiple markers",
			[]json.RawMessage{
				json.RawMessage(`{"type":"message","role":"system","content":"system prompt"}`),
				json.RawMessage(`{"type":"message","role":"system","content":"<!-- cache -->"}`),
				json.RawMessage(`{"type":"message","role":"user","content":"hello"}`),
				json.RawMessage(`{"type":"message","role":"assistant","content":"hi"}`),
				json.RawMessage(`{"type":"message","role":"system","content":"<!-- cache -->"}`),
				json.RawMessage(`{"type":"message","role":"user","content":"again"}`),
			},
			4,
			map[int]bool{0: true, 2: true},
		},
		{
			"marker at start with no preceding message",
			[]json.RawMessage{
				json.RawMessage(`{"type":"message","role":"system","content":"<!-- cache -->"}`),
				json.RawMessage(`{"type":"message","role":"user","content":"hello"}`),
			},
			1,
			nil,
		},
		{
			"marker with extra whitespace",
			[]json.RawMessage{
				json.RawMessage(`{"type":"message","role":"user","content":"hello"}`),
				json.RawMessage(`{"type":"message","role":"system","content":"<!--  cache  -->"}`),
			},
			1,
			map[int]bool{0: true},
		},
		{
			"non-system role is not a marker",
			[]json.RawMessage{
				json.RawMessage(`{"type":"message","role":"user","content":"<!-- cache -->"}`),
			},
			1,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cleaned, breakpoints := ExtractCacheBreakpoints(tc.messages)
			if len(cleaned) != tc.wantLen {
				t.Errorf("cleaned length = %d, want %d", len(cleaned), tc.wantLen)
			}
			if tc.wantBreakpoints == nil && breakpoints != nil {
				t.Errorf("breakpoints = %v, want nil", breakpoints)
			}
			if tc.wantBreakpoints != nil {
				if len(breakpoints) != len(tc.wantBreakpoints) {
					t.Errorf("breakpoints length = %d, want %d", len(breakpoints), len(tc.wantBreakpoints))
				}
				for idx := range tc.wantBreakpoints {
					if !breakpoints[idx] {
						t.Errorf("expected breakpoint at index %d", idx)
					}
				}
			}
		})
	}
}
