package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestToChatMessages(t *testing.T) {
	cases := []struct {
		name   string
		system string
		input  []string
		want   []string
	}{
		{
			name:   "user message",
			system: "You are helpful.",
			input:  []string{`{"type":"message","role":"user","content":"hello"}`},
			want: []string{
				`{"role":"system","content":"You are helpful."}`,
				`{"role":"user","content":"hello"}`,
			},
		},
		{
			name:   "function_call becomes tool_calls",
			system: "sys",
			input:  []string{`{"type":"function_call","call_id":"c1","name":"get_weather","arguments":"{\"city\":\"NYC\"}"}`},
			want: []string{
				`{"role":"system","content":"sys"}`,
				`{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"NYC\"}"}}]}`,
			},
		},
		{
			name:   "function_call_output becomes tool role",
			system: "sys",
			input:  []string{`{"type":"function_call_output","call_id":"c1","output":"sunny"}`},
			want: []string{
				`{"role":"system","content":"sys"}`,
				`{"role":"tool","tool_call_id":"c1","content":"sunny"}`,
			},
		},
		{
			name:   "system message passthrough",
			system: "sys",
			input: []string{
				`{"type":"message","role":"system","content":"extra context"}`,
				`{"type":"message","role":"user","content":"hi"}`,
			},
			want: []string{
				`{"role":"system","content":"sys"}`,
				`{"role":"system","content":"extra context"}`,
				`{"role":"user","content":"hi"}`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := toRawMessages(tc.input)
			got := toChatMessages(tc.system, input)
			gotStrings := rawMessagesToNormalized(got)
			wantStrings := normalizeJSONStrings(tc.want)
			if diff := cmp.Diff(wantStrings, gotStrings); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestForwardChatStream(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantEvents []string
		wantUsage  *UsageResponse
	}{
		{
			name: "text content delta",
			input: lines(
				`data: {"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`,
				`data: {"choices":[{"delta":{"content":" world"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
				`data: [DONE]`,
			),
			wantEvents: []string{
				`event: response.output_text.delta`,
				`event: response.output_text.delta`,
				`event: response.completed`,
			},
			wantUsage: &UsageResponse{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
		},
		{
			name: "tool call stream",
			input: lines(
				`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}`,
				`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]},"finish_reason":null}]}`,
				`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"NYC\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}}`,
				`data: [DONE]`,
			),
			wantEvents: []string{
				`event: response.output_item.added`,
				`event: response.function_call_arguments.delta`,
				`event: response.function_call_arguments.delta`,
				`event: response.output_item.done`,
				`event: response.completed`,
			},
			wantUsage: &UsageResponse{InputTokens: 20, OutputTokens: 10, TotalTokens: 30},
		},
		{
			name: "usage only chunk",
			input: lines(
				`data: {"choices":[],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`,
				`data: [DONE]`,
			),
			wantEvents: []string{
				`event: response.completed`,
			},
			wantUsage: &UsageResponse{InputTokens: 5, OutputTokens: 3, TotalTokens: 8},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := strings.NewReader(tc.input)
			rec := httptest.NewRecorder()
			flusher := rec
			usage := forwardChatStream(context.Background(), src, rec, flusher, false, "test")

			body := rec.Body.String()
			gotEvents := extractEventLines(body)

			if diff := cmp.Diff(tc.wantEvents, gotEvents); diff != "" {
				t.Errorf("events mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantUsage, usage); diff != "" {
				t.Errorf("usage mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func toRawMessages(ss []string) []json.RawMessage {
	result := make([]json.RawMessage, len(ss))
	for i, s := range ss {
		result[i] = json.RawMessage(s)
	}
	return result
}

func rawMessagesToNormalized(msgs []json.RawMessage) []string {
	result := make([]string, len(msgs))
	for i, m := range msgs {
		result[i] = normalizeJSON(string(m))
	}
	return result
}

func normalizeJSONStrings(ss []string) []string {
	result := make([]string, len(ss))
	for i, s := range ss {
		result[i] = normalizeJSON(s)
	}
	return result
}

func normalizeJSON(s string) string {
	var v any
	if json.Unmarshal([]byte(s), &v) != nil {
		return s
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
	return strings.TrimSpace(buf.String())
}

func lines(ss ...string) string {
	return strings.Join(ss, "\n") + "\n"
}

func extractEventLines(body string) []string {
	var events []string
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "event: ") {
			events = append(events, line)
		}
	}
	return events
}
