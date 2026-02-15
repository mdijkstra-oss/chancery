package http

import (
	"fmt"
	"log/slog"
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

func logUsage(endpoint string, toolNames []string, sources []string, usage *UsageResponse, pricing prompts.Pricing) {
	attrs := []any{
		"endpoint", endpoint,
		"tools", toolNames,
		"prompts", sources,
		"input_tokens", usage.InputTokens,
		"output_tokens", usage.OutputTokens,
		"total_tokens", usage.TotalTokens,
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

	slog.Info("usage", attrs...)
}
