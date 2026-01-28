package http

import (
	"math"
	"testing"

	"hermes-logos/internal/prompts"
)

func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}

func TestCalculateTokenCost(t *testing.T) {
	tests := []struct {
		name            string
		tokens          int
		centsPerMillion float64
		want            float64
	}{
		{"1M tokens at 100 cents", 1_000_000, 100, 100},
		{"500K tokens at 100 cents", 500_000, 100, 50},
		{"1471 tokens at 125 cents", 1471, 125, 0.183875},
		{"8643 tokens at 1000 cents", 8643, 1000, 8.643},
		{"1200 tokens at 12.5 cents", 1200, 12.5, 0.015},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateTokenCost(tt.tokens, tt.centsPerMillion)
			if !floatEquals(got, tt.want) {
				t.Errorf("calculateTokenCost(%d, %f) = %f, want %f", tt.tokens, tt.centsPerMillion, got, tt.want)
			}
		})
	}
}

func TestCalculateInputCost(t *testing.T) {
	pricing := prompts.Pricing{
		Input:       175,
		Output:      1400,
		CachedInput: 17.5,
	}

	tests := []struct {
		name         string
		promptTokens int
		cachedTokens int
		want         float64
	}{
		{"no cache", 1000, 0, 0.175},
		{"all cached", 1000, 1000, 0.0175},
		{"partial cache", 1471, 1200, 0.068425},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateInputCost(tt.promptTokens, tt.cachedTokens, pricing)
			if !floatEquals(got, tt.want) {
				t.Errorf("calculateInputCost(%d, %d) = %f, want %f", tt.promptTokens, tt.cachedTokens, got, tt.want)
			}
		})
	}
}

func TestCalculateUsageCost(t *testing.T) {
	pricing := prompts.Pricing{
		Input:       175,
		Output:      1400,
		CachedInput: 17.5,
	}

	tests := []struct {
		name  string
		usage UsageResponse
		want  Cost
	}{
		{
			name: "with cache",
			usage: UsageResponse{
				InputTokens:  1471,
				OutputTokens: 8643,
				TotalTokens:  10114,
				InputTokensDetails: &PromptTokensDetails{
					CachedTokens: 1200,
				},
			},
			want: Cost{
				InputCost:  0.068425,
				OutputCost: 12.1002,
				TotalCost:  12.168625,
			},
		},
		{
			name: "no cache",
			usage: UsageResponse{
				InputTokens:  1471,
				OutputTokens: 8643,
				TotalTokens:  10114,
			},
			want: Cost{
				InputCost:  0.257425,
				OutputCost: 12.1002,
				TotalCost:  12.357625,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateUsageCost(tt.usage, pricing)
			if !floatEquals(got.InputCost, tt.want.InputCost) {
				t.Errorf("InputCost = %f, want %f", got.InputCost, tt.want.InputCost)
			}
			if !floatEquals(got.OutputCost, tt.want.OutputCost) {
				t.Errorf("OutputCost = %f, want %f", got.OutputCost, tt.want.OutputCost)
			}
			if !floatEquals(got.TotalCost, tt.want.TotalCost) {
				t.Errorf("TotalCost = %f, want %f", got.TotalCost, tt.want.TotalCost)
			}
		})
	}
}
