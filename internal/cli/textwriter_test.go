package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestTextWriter(t *testing.T) {
	events := "event: response.output_text.delta\ndata: {\"delta\":\"first \"}\n\nevent: response.reasoning_summary_text.delta\ndata: {\"delta\":\"hidden\"}\n\nevent: response.output_text.delta\ndata: {\"delta\":\"second\"}\n\nevent: response.completed\ndata: {\"response\":{\"status\":\"completed\"}}\n\n"
	tests := []struct {
		name   string
		chunks []string
	}{
		{name: "single write", chunks: []string{events}},
		{name: "event writes", chunks: []string{events[:70], events[70:]}},
		{name: "partial lines", chunks: []string{events[:10], events[10:43], events[43:91], events[91:]}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			writer := newTextWriter(&output)
			for _, chunk := range test.chunks {
				if _, err := writer.Write([]byte(chunk)); err != nil {
					t.Fatalf("write: %v", err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
			if got := output.String(); got != "first second" {
				t.Errorf("output = %q, want %q", got, "first second")
			}
		})
	}
}

func TestTextWriterRejectsTruncatedStream(t *testing.T) {
	writer := newTextWriter(&bytes.Buffer{})
	if _, err := writer.Write([]byte("event: response.output_text.delta\ndata: {\"delta\":\"partial\"}\n\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	err := writer.Close()
	if err == nil || !strings.Contains(err.Error(), "before response.completed") {
		t.Fatalf("close error = %v", err)
	}
}

func TestTextWriterReturnsStreamFailure(t *testing.T) {
	event := "event: response.failed\ndata: {\"response\":{\"status\":\"failed\",\"error\":{\"type\":\"limit\",\"message\":\"output truncated\"}}}\n\n"
	writer := newTextWriter(&bytes.Buffer{})
	if _, err := writer.Write([]byte(event)); err != nil {
		t.Fatalf("write: %v", err)
	}
	err := writer.Close()
	if err == nil || !strings.Contains(err.Error(), "output truncated") {
		t.Fatalf("close error = %v", err)
	}
}

func TestOutputText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "text deltas",
			input: "event: response.output_text.delta\ndata: {\"delta\":\"hello \"}\n\nevent: response.output_text.delta\ndata: {\"delta\":\"world\"}\n\n",
			want:  "hello world",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "reasoning events ignored",
			input: "event: response.reasoning_summary_text.delta\ndata: {\"delta\":\"internal thought\"}\n\nevent: response.output_text.delta\ndata: {\"delta\":\"visible\"}\n\n",
			want:  "visible",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := outputText(strings.Split(test.input, "\n")); got != test.want {
				t.Errorf("outputText() = %q, want %q", got, test.want)
			}
		})
	}
}
