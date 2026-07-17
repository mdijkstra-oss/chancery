package sse

import "testing"

func TestTextDeltaEvent(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"plain", "hello", `{"delta":"hello"}`},
		{"empty", "", `{"delta":""}`},
		{"escapes", "a\"b\n", `{"delta":"a\"b\n"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TextDeltaEvent(tt.text)
			if got.Type != "response.output_text.delta" {
				t.Errorf("Type = %q, want response.output_text.delta", got.Type)
			}
			if got.Data != tt.want {
				t.Errorf("Data = %q, want %q", got.Data, tt.want)
			}
		})
	}
}

func TestReasoningDeltaEvent(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"plain", "thinking", `{"delta":"thinking"}`},
		{"empty", "", `{"delta":""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReasoningDeltaEvent(tt.text)
			if got.Type != "response.reasoning_summary_text.delta" {
				t.Errorf("Type = %q, want response.reasoning_summary_text.delta", got.Type)
			}
			if got.Data != tt.want {
				t.Errorf("Data = %q, want %q", got.Data, tt.want)
			}
		})
	}
}

type fakeReason string

func TestFinishReasonToEvent(t *testing.T) {
	errs := map[fakeReason]string{
		"length": "output truncated: token limit reached",
	}
	tests := []struct {
		name       string
		reason     fakeReason
		wantNil    bool
		wantType   string
		wantErrMsg string
	}{
		{"known", "length", false, "length", "output truncated: token limit reached"},
		{"unknown", "stop", true, "", ""},
		{"empty", "", true, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := FinishReasonToEvent(errs, tt.reason)
			if tt.wantNil {
				if event != nil {
					t.Errorf("event = %+v, want nil", event)
				}
				return
			}
			if event == nil {
				t.Fatalf("event = nil, want failed event")
			}
			if event.Type != "response.failed" {
				t.Errorf("Type = %q, want response.failed", event.Type)
			}
			want := BuildFailedEvent(tt.wantType, tt.wantErrMsg)
			if event.Data != want.Data {
				t.Errorf("Data = %q, want %q", event.Data, want.Data)
			}
		})
	}
}
