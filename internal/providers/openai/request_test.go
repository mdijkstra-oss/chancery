package openai

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/matthijn/hermes-logos/internal/prompts"
	"github.com/matthijn/hermes-logos/internal/protocol"
)

func TestBuildHTTPRequest(t *testing.T) {
	params := protocol.RequestParams{
		Model:        "gpt-4o",
		SystemPrompt: "you are helpful",
		Messages:     []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi"}`)},
	}
	provider := prompts.ProviderConfig{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-test-key",
	}

	req, err := BuildHTTPRequest(context.Background(), params, provider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			name: "url",
			check: func(t *testing.T) {
				want := "https://api.openai.com/v1/responses"
				if req.URL.String() != want {
					t.Errorf("url = %q, want %q", req.URL.String(), want)
				}
			},
		},
		{
			name: "method",
			check: func(t *testing.T) {
				if req.Method != "POST" {
					t.Errorf("method = %q, want POST", req.Method)
				}
			},
		},
		{
			name: "content type header",
			check: func(t *testing.T) {
				got := req.Header.Get("Content-Type")
				if got != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", got)
				}
			},
		},
		{
			name: "authorization header",
			check: func(t *testing.T) {
				got := req.Header.Get("Authorization")
				if got != "Bearer sk-test-key" {
					t.Errorf("Authorization = %q, want Bearer sk-test-key", got)
				}
			},
		},
		{
			name: "body contains model",
			check: func(t *testing.T) {
				body, _ := io.ReadAll(req.Body)
				var parsed map[string]json.RawMessage
				if err := json.Unmarshal(body, &parsed); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				var model string
				json.Unmarshal(parsed["model"], &model)
				if model != "gpt-4o" {
					t.Errorf("model = %q, want gpt-4o", model)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t)
		})
	}
}
