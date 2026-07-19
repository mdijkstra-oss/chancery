package sse

import (
	"strings"
	"testing"
)

func TestNewScannerReadsLinesAboveDefaultBufferLimit(t *testing.T) {
	longData := strings.Repeat("x", 200*1024)
	input := "event: response.completed\ndata: " + longData + "\n"

	scanner := NewScanner(strings.NewReader(input))

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan error on long line: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if got := len(lines[1]); got != len("data: ")+len(longData) {
		t.Fatalf("long data line truncated: got %d bytes", got)
	}
}

func TestEventAndDataField(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantEvent string
		wantEvOK  bool
		wantData  string
		wantDaOK  bool
	}{
		{name: "event line", line: "event: message_delta", wantEvent: "message_delta", wantEvOK: true},
		{name: "data line", line: "data: {\"x\":1}", wantData: "{\"x\":1}", wantDaOK: true},
		{name: "blank line", line: ""},
		{name: "comment line", line: ": keep-alive"},
		{name: "no space prefix", line: "data:no-space"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, evOK := EventField(tt.line)
			if ev != tt.wantEvent || evOK != tt.wantEvOK {
				t.Fatalf("EventField(%q) = (%q, %v), want (%q, %v)", tt.line, ev, evOK, tt.wantEvent, tt.wantEvOK)
			}
			da, daOK := DataField(tt.line)
			if da != tt.wantData || daOK != tt.wantDaOK {
				t.Fatalf("DataField(%q) = (%q, %v), want (%q, %v)", tt.line, da, daOK, tt.wantData, tt.wantDaOK)
			}
		})
	}
}
