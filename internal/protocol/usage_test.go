package protocol

import "testing"

func TestUsageSelectors(t *testing.T) {
	tests := []struct {
		name           string
		usage          *UsageResponse
		wantCached     int
		wantCacheWrite int
		wantReasoning  int
	}{
		{name: "nil"},
		{name: "empty", usage: &UsageResponse{}},
		{
			name: "details",
			usage: &UsageResponse{
				InputTokensDetails:  &PromptTokensDetails{CachedTokens: 10, CacheCreationTokens: 20},
				OutputTokensDetails: &OutputTokensDetails{ReasoningTokens: 30},
			},
			wantCached:     10,
			wantCacheWrite: 20,
			wantReasoning:  30,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CachedInputTokens(test.usage); got != test.wantCached {
				t.Errorf("CachedInputTokens() = %d, want %d", got, test.wantCached)
			}
			if got := CacheWriteTokens(test.usage); got != test.wantCacheWrite {
				t.Errorf("CacheWriteTokens() = %d, want %d", got, test.wantCacheWrite)
			}
			if got := ReasoningTokens(test.usage); got != test.wantReasoning {
				t.Errorf("ReasoningTokens() = %d, want %d", got, test.wantReasoning)
			}
		})
	}
}
