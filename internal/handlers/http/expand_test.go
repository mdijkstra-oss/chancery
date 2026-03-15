package http

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"hermes-logos/internal/prompts"
)

func msg(role, content string) json.RawMessage {
	b, _ := json.Marshal(InputMessage{Role: role, Content: content})
	return b
}

func typedMsg(role, content string) json.RawMessage {
	b, _ := json.Marshal(InputMessage{Type: "message", Role: role, Content: content})
	return b
}

func TestExpandMessages(t *testing.T) {
	modes := map[string]string{
		"planning":  "You are in planning mode.",
		"execution": "You are in execution mode.",
	}

	cases := []struct {
		name     string
		messages []json.RawMessage
		want     []json.RawMessage
	}{
		{
			name:     "planning marker expanded",
			messages: []json.RawMessage{msg("system", "<!-- prompt: planning -->")},
			want:     []json.RawMessage{msg("system", "You are in planning mode.")},
		},
		{
			name:     "execution marker expanded",
			messages: []json.RawMessage{msg("system", "<!-- prompt: execution -->")},
			want:     []json.RawMessage{msg("system", "You are in execution mode.")},
		},
		{
			name:     "regular system message unchanged",
			messages: []json.RawMessage{msg("system", "You are helpful.")},
			want:     []json.RawMessage{msg("system", "You are helpful.")},
		},
		{
			name:     "user message unchanged",
			messages: []json.RawMessage{msg("user", "<!-- prompt: planning -->")},
			want:     []json.RawMessage{msg("user", "<!-- prompt: planning -->")},
		},
		{
			name:     "function call passthrough",
			messages: []json.RawMessage{json.RawMessage(`{"type":"function_call","name":"test","arguments":"{}"}`)},
			want:     []json.RawMessage{json.RawMessage(`{"type":"function_call","name":"test","arguments":"{}"}`)},
		},
		{
			name:     "unknown mode unchanged",
			messages: []json.RawMessage{msg("system", "<!-- prompt: nonexistent -->")},
			want:     []json.RawMessage{msg("system", "<!-- prompt: nonexistent -->")},
		},
		{
			name: "multiple messages one marker",
			messages: []json.RawMessage{
				msg("user", "Hello"),
				msg("system", "<!-- prompt: planning -->"),
				msg("user", "Let's plan"),
			},
			want: []json.RawMessage{
				msg("user", "Hello"),
				msg("system", "You are in planning mode."),
				msg("user", "Let's plan"),
			},
		},
		{
			name:     "marker with extra whitespace",
			messages: []json.RawMessage{msg("system", "<!--  prompt:  planning  -->")},
			want:     []json.RawMessage{msg("system", "You are in planning mode.")},
		},
		{
			name: "multiple markers last wins",
			messages: []json.RawMessage{
				msg("user", "Hello"),
				msg("system", "<!-- prompt: planning -->"),
				msg("user", "Now execute"),
				msg("system", "<!-- prompt: execution -->"),
			},
			want: []json.RawMessage{
				msg("user", "Hello"),
				msg("user", "Now execute"),
				msg("system", "You are in execution mode."),
			},
		},
		{
			name: "three markers last wins earlier dropped",
			messages: []json.RawMessage{
				msg("system", "<!-- prompt: execution -->"),
				msg("user", "First"),
				msg("system", "<!-- prompt: planning -->"),
				msg("user", "Second"),
				msg("system", "<!-- prompt: execution -->"),
				msg("user", "Third"),
			},
			want: []json.RawMessage{
				msg("user", "First"),
				msg("user", "Second"),
				msg("system", "You are in execution mode."),
				msg("user", "Third"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExpandMessages(tc.messages, modes)
			if diff := cmp.Diff(rawSliceToStrings(tc.want), rawSliceToStrings(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractReasoningEffort(t *testing.T) {
	cases := []struct {
		name         string
		messages     []json.RawMessage
		wantEffort   string
		wantMsgCount int
	}{
		{
			name:         "no markers",
			messages:     []json.RawMessage{msg("user", "hello")},
			wantEffort:   "",
			wantMsgCount: 1,
		},
		{
			name:         "single marker",
			messages:     []json.RawMessage{msg("system", "<!-- reasoning: high -->")},
			wantEffort:   "high",
			wantMsgCount: 0,
		},
		{
			name: "last marker wins",
			messages: []json.RawMessage{
				msg("system", "<!-- reasoning: low -->"),
				msg("user", "hi"),
				msg("system", "<!-- reasoning: high -->"),
			},
			wantEffort:   "high",
			wantMsgCount: 1,
		},
		{
			name: "markers stripped other messages kept",
			messages: []json.RawMessage{
				msg("user", "hello"),
				msg("system", "<!-- reasoning: medium -->"),
				msg("user", "go"),
			},
			wantEffort:   "medium",
			wantMsgCount: 2,
		},
		{
			name:         "non-system reasoning marker unchanged",
			messages:     []json.RawMessage{msg("user", "<!-- reasoning: high -->")},
			wantEffort:   "",
			wantMsgCount: 1,
		},
		{
			name:         "marker with extra whitespace",
			messages:     []json.RawMessage{msg("system", "<!--  reasoning:  medium  -->")},
			wantEffort:   "medium",
			wantMsgCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotMessages, gotEffort := ExtractReasoningEffort(tc.messages)
			if gotEffort != tc.wantEffort {
				t.Errorf("effort: got %q, want %q", gotEffort, tc.wantEffort)
			}
			if len(gotMessages) != tc.wantMsgCount {
				t.Errorf("message count: got %d, want %d", len(gotMessages), tc.wantMsgCount)
			}
		})
	}
}

func TestExtractVerbosity(t *testing.T) {
	cases := []struct {
		name          string
		messages      []json.RawMessage
		wantVerbosity string
		wantMsgCount  int
	}{
		{
			name:          "no markers",
			messages:      []json.RawMessage{msg("user", "hello")},
			wantVerbosity: "",
			wantMsgCount:  1,
		},
		{
			name:          "single marker",
			messages:      []json.RawMessage{msg("system", "<!-- verbosity: medium -->")},
			wantVerbosity: "medium",
			wantMsgCount:  0,
		},
		{
			name: "last marker wins",
			messages: []json.RawMessage{
				msg("system", "<!-- verbosity: low -->"),
				msg("user", "hi"),
				msg("system", "<!-- verbosity: medium -->"),
			},
			wantVerbosity: "medium",
			wantMsgCount:  1,
		},
		{
			name: "markers stripped other messages kept",
			messages: []json.RawMessage{
				msg("user", "hello"),
				msg("system", "<!-- verbosity: low -->"),
				msg("user", "go"),
			},
			wantVerbosity: "low",
			wantMsgCount:  2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotMessages, gotVerbosity := ExtractVerbosity(tc.messages)
			if gotVerbosity != tc.wantVerbosity {
				t.Errorf("verbosity: got %q, want %q", gotVerbosity, tc.wantVerbosity)
			}
			if len(gotMessages) != tc.wantMsgCount {
				t.Errorf("message count: got %d, want %d", len(gotMessages), tc.wantMsgCount)
			}
		})
	}
}

func TestExpandApproaches(t *testing.T) {
	entries := map[string]prompts.Approach{
		"index":       {Key: "index", Content: "## Root"},
		"a/index":     {Key: "a/index", Content: "## A Index"},
		"a/b/index":   {Key: "a/b/index", Content: "## A/B Index"},
		"a/b/leaf":    {Key: "a/b/leaf", Content: "## Leaf content"},
		"a/b/another": {Key: "a/b/another", Content: "## Another content"},
	}

	cases := []struct {
		name     string
		messages []json.RawMessage
		want     []json.RawMessage
	}{
		{
			name:     "single marker expanded with indexes",
			messages: []json.RawMessage{msg("system", "<!-- approach: a/b/leaf -->")},
			want: []json.RawMessage{
				typedMsg("system", "[a/b/index]\n## A/B Index"),
				typedMsg("system", "[a/index]\n## A Index"),
				typedMsg("system", "[index]\n## Root"),
				typedMsg("system", "[a/b/leaf]\n## Leaf content"),
			},
		},
		{
			name: "multiple markers all expanded",
			messages: []json.RawMessage{
				msg("system", "<!-- approach: a/b/leaf -->"),
				msg("user", "hello"),
				msg("system", "<!-- approach: a/b/another -->"),
			},
			want: []json.RawMessage{
				typedMsg("system", "[a/b/index]\n## A/B Index"),
				typedMsg("system", "[a/index]\n## A Index"),
				typedMsg("system", "[index]\n## Root"),
				typedMsg("system", "[a/b/leaf]\n## Leaf content"),
				msg("user", "hello"),
				typedMsg("system", "[a/b/another]\n## Another content"),
			},
		},
		{
			name:     "unknown key dropped known index kept",
			messages: []json.RawMessage{msg("system", "<!-- approach: unknown/key -->")},
			want: []json.RawMessage{
				typedMsg("system", "[index]\n## Root"),
			},
		},
		{
			name: "no markers passthrough",
			messages: []json.RawMessage{
				msg("user", "hello"),
				msg("system", "some system msg"),
			},
			want: []json.RawMessage{
				msg("user", "hello"),
				msg("system", "some system msg"),
			},
		},
		{
			name: "mixed modes and approaches both work",
			messages: []json.RawMessage{
				msg("system", "<!-- prompt: planning -->"),
				msg("system", "<!-- approach: a/b/leaf -->"),
			},
			want: []json.RawMessage{
				msg("system", "<!-- prompt: planning -->"),
				typedMsg("system", "[a/b/index]\n## A/B Index"),
				typedMsg("system", "[a/index]\n## A Index"),
				typedMsg("system", "[index]\n## Root"),
				typedMsg("system", "[a/b/leaf]\n## Leaf content"),
			},
		},
		{
			name:     "non-system approach marker unchanged",
			messages: []json.RawMessage{msg("user", "<!-- approach: a/b/leaf -->")},
			want:     []json.RawMessage{msg("user", "<!-- approach: a/b/leaf -->")},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExpandApproaches(tc.messages, entries)
			if diff := cmp.Diff(rawSliceToStrings(tc.want), rawSliceToStrings(got)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func rawSliceToStrings(raws []json.RawMessage) []string {
	result := make([]string, len(raws))
	for i, r := range raws {
		result[i] = string(r)
	}
	return result
}
