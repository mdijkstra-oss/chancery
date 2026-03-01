package http

import (
	"bytes"
	"encoding/json"
	"strings"
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
			got := stripForCompaction(tc.messages)
			if len(got) != tc.expected {
				t.Errorf("stripForCompaction() returned %d messages, want %d", len(got), tc.expected)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		name     string
		input    []json.RawMessage
		expected int
	}{
		{"empty input", nil, 0},
		{"single message", []json.RawMessage{json.RawMessage(`{"role":"user","content":"hello"}`)}, 8},
		{"multiple messages", []json.RawMessage{
			json.RawMessage(`{"role":"user","content":"hi"}`),
			json.RawMessage(`{"role":"assistant","content":"hello there"}`),
		}, 18},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := estimateTokens(tc.input)
			if got != tc.expected {
				t.Errorf("estimateTokens() = %d, want %d", got, tc.expected)
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
			got := shouldCompact(tc.compactAt, tc.tokens)
			if got != tc.expected {
				t.Errorf("shouldCompact(%d, %d) = %v, want %v", tc.compactAt, tc.tokens, got, tc.expected)
			}
		})
	}
}

func TestBuildCompactedDoneData(t *testing.T) {
	cases := []struct {
		name    string
		summary string
	}{
		{"simple summary", "The user discussed project setup."},
		{"summary with quotes", `User said "hello" and asked about "plans".`},
		{"summary with newlines", "Line one.\nLine two.\nLine three."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := buildCompactedDoneData(tc.summary)

			var parsed struct {
				Item struct {
					Type      string `json:"type"`
					CallID    string `json:"call_id"`
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"item"`
			}
			if err := json.Unmarshal([]byte(data), &parsed); err != nil {
				t.Fatalf("failed to parse done data: %v", err)
			}

			if parsed.Item.Type != "function_call" {
				t.Errorf("type = %q, want function_call", parsed.Item.Type)
			}
			if parsed.Item.Name != "compacted" {
				t.Errorf("name = %q, want compacted", parsed.Item.Name)
			}

			var args compactedArgs
			if err := json.Unmarshal([]byte(parsed.Item.Arguments), &args); err != nil {
				t.Fatalf("failed to parse arguments: %v", err)
			}
			if args.Summary != tc.summary {
				t.Errorf("summary = %q, want %q", args.Summary, tc.summary)
			}
		})
	}
}

type testFlusher struct {
	flushed int
}

func (f *testFlusher) Flush() { f.flushed++ }

func TestStreamCompaction(t *testing.T) {
	cases := []struct {
		name            string
		sseInput        string
		expectedSummary string
		expectedEvents  []string
		hasUsage        bool
	}{
		{
			name: "single text delta",
			sseInput: strings.Join([]string{
				"event: response.output_text.delta",
				`data: {"delta":"Hello world"}`,
				"",
				"event: response.completed",
				`data: {"response":{"usage":{"input_tokens":100,"output_tokens":50,"total_tokens":150}}}`,
				"",
			}, "\n"),
			expectedSummary: "Hello world",
			expectedEvents:  []string{"response.output_item.added", "response.function_call_arguments.delta", "response.output_item.done", "response.completed"},
			hasUsage:        true,
		},
		{
			name: "multiple text deltas",
			sseInput: strings.Join([]string{
				"event: response.output_text.delta",
				`data: {"delta":"Part one. "}`,
				"",
				"event: response.output_text.delta",
				`data: {"delta":"Part two."}`,
				"",
				"event: response.completed",
				`data: {"response":{}}`,
				"",
			}, "\n"),
			expectedSummary: "Part one. Part two.",
			expectedEvents:  []string{"response.output_item.added", "response.function_call_arguments.delta", "response.function_call_arguments.delta", "response.output_item.done", "response.completed"},
			hasUsage:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := strings.NewReader(tc.sseInput)
			var buf bytes.Buffer
			flusher := &testFlusher{}

			type writerFlusher struct {
				*bytes.Buffer
				*testFlusher
			}
			wf := writerFlusher{&buf, flusher}

			usage, err := streamCompaction(src, wf.Buffer, wf.testFlusher)
			if err != nil {
				t.Fatalf("streamCompaction error: %v", err)
			}

			output := buf.String()

			for _, event := range tc.expectedEvents {
				if !strings.Contains(output, "event: "+event) {
					t.Errorf("missing event %q in output", event)
				}
			}

			if tc.hasUsage && usage == nil {
				t.Error("expected usage, got nil")
			}

			if !strings.Contains(output, tc.expectedSummary) {
				t.Errorf("summary %q not found in output:\n%s", tc.expectedSummary, output)
			}

			doneIdx := strings.Index(output, "event: response.output_item.done")
			if doneIdx == -1 {
				t.Fatal("missing response.output_item.done event")
			}
			doneData := output[doneIdx:]
			if !strings.Contains(doneData, `"name":"compacted"`) {
				t.Error("done event missing compacted name")
			}
		})
	}
}
