package http

import (
	"context"
	"encoding/json"
	"testing"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/quota"
)

func TestBuildQuotaRequest(t *testing.T) {
	tests := []struct {
		name string
		got  quota.ReserveRequest
		want quota.ReserveRequest
	}{
		{
			name: "authenticated chat",
			got: buildChatQuotaRequest(
				"req-1",
				"user-1",
				"agent",
				protocol.RequestParams{
					Model:           "model-a",
					SystemPrompt:    "12345678",
					Messages:        []json.RawMessage{json.RawMessage(`12345678`)},
					Tools:           []json.RawMessage{json.RawMessage(`1234`)},
					MaxTokens:       100,
					ServiceTier:     "priority",
					ReasoningEffort: "high",
				},
				prompts.PromptConfig{Provider: prompts.ProviderConfig{Key: "provider-a"}},
			),
			want: quota.ReserveRequest{RequestID: "req-1", Subject: "user-1", Operation: "chat", Endpoint: "agent", Provider: "provider-a", Model: "model-a", ServiceTier: "priority", ReasoningEffort: "high", EstimatedInputTokens: 5, MaximumOutputTokens: 100},
		},
		{
			name: "anonymous embeddings",
			got:  buildEmbeddingsQuotaRequest("req-2", "", []string{"abcdefgh"}, prompts.PromptConfig{Model: "embedding-a", Provider: prompts.ProviderConfig{Key: "provider-b"}}),
			want: quota.ReserveRequest{RequestID: "req-2", Operation: "embeddings", Endpoint: "embeddings", Provider: "provider-b", Model: "embedding-a", EstimatedInputTokens: 2},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("request = %#v, want %#v", test.got, test.want)
			}
		})
	}
}

func TestQuotaUsage(t *testing.T) {
	tests := []struct {
		name  string
		input *protocol.UsageResponse
		want  *quota.Usage
	}{
		{name: "nil"},
		{
			name: "normalized",
			input: &protocol.UsageResponse{
				InputTokens:         100,
				OutputTokens:        50,
				TotalTokens:         150,
				InputTokensDetails:  &protocol.PromptTokensDetails{CachedTokens: 20, CacheCreationTokens: 10},
				OutputTokensDetails: &protocol.OutputTokensDetails{ReasoningTokens: 30},
			},
			want: &quota.Usage{InputTokens: 100, CachedInputTokens: 20, CacheWriteTokens: 10, OutputTokens: 50, ReasoningTokens: 30, TotalTokens: 150},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := quotaUsage(test.input)
			if got == nil || test.want == nil {
				if got != test.want {
					t.Errorf("usage = %#v, want %#v", got, test.want)
				}
				return
			}
			if *got != *test.want {
				t.Errorf("usage = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestFailedQuotaOutcome(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name string
		ctx  context.Context
		want quota.Outcome
	}{
		{name: "failed", ctx: context.Background(), want: quota.OutcomeFailed},
		{name: "cancelled", ctx: cancelled, want: quota.OutcomeCancelled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := failedQuotaOutcome(test.ctx); got != test.want {
				t.Errorf("outcome = %q, want %q", got, test.want)
			}
		})
	}
}
