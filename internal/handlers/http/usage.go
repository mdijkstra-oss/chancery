package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
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

func formatRatio(prompt, completion int) string {
	if completion == 0 {
		return "0:1"
	}
	ratio := math.Round(float64(prompt) / float64(completion))
	return fmt.Sprintf("%.0f:1", ratio)
}

func buildUsageAttrs(usage *UsageResponse, breakdown TokenBreakdown, breakpoints []CacheBreakpointInfo, enableCaching, hasTools bool) []any {
	ratio := formatRatio(usage.PromptTokens, usage.CompletionTokens)
	totalEstimate := sumTokens(breakdown)
	estCached := calculateCachedTokens(breakdown, breakpoints, enableCaching, hasTools)
	estUncached := totalEstimate - estCached

	attrs := []any{
		"prompt_tokens", usage.PromptTokens,
		"completion_tokens", usage.CompletionTokens,
		"total_tokens", usage.TotalTokens,
		"input_output_ratio", ratio,
		"cache_discount", usage.CacheDiscount,
		"est_system", breakdown.System,
		"est_tool_defs", breakdown.ToolDefs,
		"est_user_msgs", breakdown.UserMsgs,
		"est_assistant_msgs", breakdown.AssistantMsgs,
		"est_tool_calls", breakdown.ToolCalls,
		"est_tool_responses", breakdown.ToolResponses,
		"est_total", totalEstimate,
		"est_cached", estCached,
		"est_uncached", estUncached,
	}

	if enableCaching && hasTools {
		attrs = append(attrs, "bp1", "tools")
	}

	for _, bp := range breakpoints {
		attrs = append(attrs, fmt.Sprintf("bp%d", bp.BreakpointNum), bp.TokenPos)
	}

	return attrs
}

func logUsage(usage *UsageResponse, breakdown TokenBreakdown, breakpoints []CacheBreakpointInfo, enableCaching, hasTools bool) {
	attrs := buildUsageAttrs(usage, breakdown, breakpoints, enableCaching, hasTools)
	slog.Info("usage", attrs...)
}

func buildRequestLogAttrs(req OpenAIRequest, breakdown TokenBreakdown) []any {
	totalEstimate := sumTokens(breakdown)
	return []any{
		"model", req.Model,
		"message_count", len(req.Messages),
		"tool_count", len(req.Tools),
		"stream", req.Stream,
		"est_system", breakdown.System,
		"est_tool_defs", breakdown.ToolDefs,
		"est_user_msgs", breakdown.UserMsgs,
		"est_assistant_msgs", breakdown.AssistantMsgs,
		"est_tool_calls", breakdown.ToolCalls,
		"est_tool_responses", breakdown.ToolResponses,
		"est_total", totalEstimate,
	}
}

func logOutgoingRequest(req OpenAIRequest, breakdown TokenBreakdown, verbose bool) {
	attrs := buildRequestLogAttrs(req, breakdown)
	slog.Info("outgoing_request", attrs...)

	if verbose {
		reqJSON, _ := json.Marshal(req)
		slog.Info("raw_outgoing_request", "data", string(reqJSON))
	}
}
