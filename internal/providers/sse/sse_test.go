package sse

import (
	"net/http/httptest"
	"testing"
)

func TestSetHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	SetHeaders(w)

	tests := []struct {
		header string
		want   string
	}{
		{"Content-Type", "text/event-stream"},
		{"Cache-Control", "no-cache"},
		{"Connection", "keep-alive"},
	}
	for _, tt := range tests {
		got := w.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("header %s = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestWriteEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		data      string
		want      string
	}{
		{
			name:      "simple text event",
			eventType: "response.output_text.delta",
			data:      `{"delta":"hello"}`,
			want:      "event: response.output_text.delta\ndata: {\"delta\":\"hello\"}\n\n",
		},
		{
			name:      "completed event",
			eventType: "response.completed",
			data:      `{"response":{"usage":{}}}`,
			want:      "event: response.completed\ndata: {\"response\":{\"usage\":{}}}\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteEvent(w, tt.eventType, tt.data)
			got := w.Body.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFlush(t *testing.T) {
	w := httptest.NewRecorder()
	Flush(w)
	if !w.Flushed {
		t.Error("expected Flush to be called on http.Flusher")
	}
}
