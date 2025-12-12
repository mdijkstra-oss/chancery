package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

func formatCost(centsCost float64) string {
	dollars := centsCost / 100.0
	return fmt.Sprintf("%.4f", dollars)
}

func extractUsage(line string) *UsageResponse {
	if !strings.HasPrefix(line, "data: ") {
		return nil
	}
	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return nil
	}
	var chunk StreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil
	}
	return chunk.Usage
}

func logUsage(usage *UsageResponse, pricing Pricing) {
	attrs := []any{
		"prompt_tokens", usage.PromptTokens,
		"completion_tokens", usage.CompletionTokens,
		"total_tokens", usage.TotalTokens,
	}
	if usage.PromptTokensDetails != nil {
		attrs = append(attrs, "cached_tokens", usage.PromptTokensDetails.CachedTokens)
	}

	cost := calculateUsageCost(pricing, *usage)
	attrs = append(attrs,
		"input_cost", formatCost(cost.InputCost),
		"output_cost", formatCost(cost.OutputCost),
		"total_cost", formatCost(cost.TotalCost),
	)

	slog.Info("usage", attrs...)
}
