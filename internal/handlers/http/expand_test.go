package http

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func msg(role, content string) json.RawMessage {
	b, _ := json.Marshal(InputMessage{Role: role, Content: content})
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

func rawSliceToStrings(raws []json.RawMessage) []string {
	result := make([]string, len(raws))
	for i, r := range raws {
		result[i] = string(r)
	}
	return result
}
