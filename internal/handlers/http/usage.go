package http

import (
	"fmt"
	"log/slog"
)

func formatCost(centsCost float64) string {
	dollars := centsCost / 100.0
	return fmt.Sprintf("%.4f", dollars)
}

func logUsage(usage *UsageResponse, pricing Pricing) {
	attrs := []any{
		"input_tokens", usage.InputTokens,
		"output_tokens", usage.OutputTokens,
		"total_tokens", usage.TotalTokens,
	}
	if usage.InputTokensDetails != nil {
		attrs = append(attrs, "cached_tokens", usage.InputTokensDetails.CachedTokens)
	}

	cost := calculateUsageCost(pricing, *usage)
	attrs = append(attrs,
		"input_cost", formatCost(cost.InputCost),
		"output_cost", formatCost(cost.OutputCost),
		"total_cost", formatCost(cost.TotalCost),
	)

	slog.Info("usage", attrs...)
}
