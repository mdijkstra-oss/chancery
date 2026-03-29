package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"

	"hermes-logos/internal/prompts"
)

var sessionCostCents atomic.Uint64

func formatCost(centsCost float64) string {
	dollars := centsCost / 100.0
	return fmt.Sprintf("%.4f", dollars)
}

func addToSessionCost(cents float64) float64 {
	centsMicro := uint64(cents * 1_000_000)
	newTotal := sessionCostCents.Add(centsMicro)
	return float64(newTotal) / 1_000_000
}

type RateLimitInfo struct {
	RemainingRequests string
	RemainingTokens   string
	ResetRequests     string
	ResetTokens       string
}

func logUsage(endpoint string, usage *UsageResponse, pricing prompts.Pricing, reasoningEffort string, estimatedTokens int, rateLimit RateLimitInfo) {
	attrs := []any{
		"endpoint", endpoint,
		"input_tokens", usage.InputTokens,
		"output_tokens", usage.OutputTokens,
		"total_tokens", usage.TotalTokens,
		"estimated_tokens", estimatedTokens,
		"reasoning", reasoningEffort,
	}
	if usage.InputTokensDetails != nil {
		attrs = append(attrs, "cached_tokens", usage.InputTokensDetails.CachedTokens)
	}

	cost := calculateUsageCost(*usage, pricing)
	sessionCost := addToSessionCost(cost.TotalCost)
	attrs = append(attrs,
		"input_cost", formatCost(cost.InputCost),
		"output_cost", formatCost(cost.OutputCost),
		"total_cost", formatCost(cost.TotalCost),
		"session_cost", formatCost(sessionCost),
	)

	if rateLimit.RemainingRequests != "" {
		attrs = append(attrs, "rpm_remaining", rateLimit.RemainingRequests)
	}
	if rateLimit.RemainingTokens != "" {
		attrs = append(attrs, "tpm_remaining", rateLimit.RemainingTokens)
	}
	if rateLimit.ResetRequests != "" {
		attrs = append(attrs, "rpm_reset", rateLimit.ResetRequests)
	}
	if rateLimit.ResetTokens != "" {
		attrs = append(attrs, "tpm_reset", rateLimit.ResetTokens)
	}

	slog.Info("usage", attrs...)
}

func extractRateLimitInfo(h http.Header) RateLimitInfo {
	return RateLimitInfo{
		RemainingRequests: h.Get("x-ratelimit-remaining-requests"),
		RemainingTokens:   h.Get("x-ratelimit-remaining-tokens"),
		ResetRequests:     h.Get("x-ratelimit-reset-requests"),
		ResetTokens:       h.Get("x-ratelimit-reset-tokens"),
	}
}
