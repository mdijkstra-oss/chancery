package sse

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
			writer := NewTextWriter(&output)
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
	writer := NewTextWriter(&bytes.Buffer{})
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
	writer := NewTextWriter(&bytes.Buffer{})
	if _, err := writer.Write([]byte(event)); err != nil {
		t.Fatalf("write: %v", err)
	}
	err := writer.Close()
	if err == nil || !strings.Contains(err.Error(), "output truncated") {
		t.Fatalf("close error = %v", err)
	}
}
