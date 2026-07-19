package messages

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/matthijn/hermes-logos/internal/protocol"
)

func msg(role, content string) json.RawMessage {
	b, _ := json.Marshal(protocol.InputMessage{Role: role, Content: content})
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

func rawSliceToStrings(raws []json.RawMessage) []string {
	result := make([]string, len(raws))
	for i, r := range raws {
		result[i] = string(r)
	}
	return result
}
