package http

import (
	"encoding/json"
	"log/slog"
	"strings"
)

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

func logUsage(usage *UsageResponse) {
	attrs := []any{
		"prompt_tokens", usage.PromptTokens,
		"completion_tokens", usage.CompletionTokens,
		"total_tokens", usage.TotalTokens,
	}
	if usage.PromptTokensDetails != nil {
		attrs = append(attrs, "cached_tokens", usage.PromptTokensDetails.CachedTokens)
	}
	slog.Info("usage", attrs...)
}
