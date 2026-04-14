package telemetry

import (
	"context"
	"encoding/json"
	"log/slog"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
)

type CallRecord struct {
	Endpoint          string
	Model             string
	Reasoning         string
	ServiceTier       string
	Trigger           string
	InputTokens       int
	CachedInputTokens int
	OutputTokens      int
	ReasoningTokens   int
	InputCost         float64
	CachedInputCost   float64
	OutputCost        float64
	ReasoningCost     float64
	TotalCost         float64
	DurationMs        int64
	InputCount        int
}

// Both OpenAI and Gemini report input tokens inclusive of cached tokens:
//   OpenAI: prompt_tokens includes cached_tokens (subset in prompt_tokens_details)
//   Gemini: PromptTokenCount "includes the number of tokens in the cached content"
func computeCosts(input, cached, output int, pricing prompts.Pricing) (float64, float64, float64) {
	uncached := input - cached
	inputCost := float64(uncached) * pricing.Input / 1_000_000
	cachedCost := float64(cached) * pricing.CachedInput / 1_000_000
	outputCost := float64(output) * pricing.Output / 1_000_000
	return inputCost, cachedCost, outputCost
}

type triggerPeek struct {
	Type   string `json:"type"`
	Role   string `json:"role"`
	CallID string `json:"call_id"`
	Name   string `json:"name"`
}

func isSystemMessage(p triggerPeek) bool {
	return p.Role == "system"
}

func isUserMessage(p triggerPeek) bool {
	return p.Role == "user"
}

func isFunctionCallOutput(p triggerPeek) bool {
	return p.Type == "function_call_output"
}

func isFunctionCall(p triggerPeek) bool {
	return p.Type == "function_call"
}

func findToolName(messages []json.RawMessage, callID string, startIdx int) string {
	for i := startIdx; i >= 0; i-- {
		var p triggerPeek
		if json.Unmarshal(messages[i], &p) != nil {
			continue
		}
		if isFunctionCall(p) && p.CallID == callID {
			return p.Name
		}
	}
	return ""
}

func DeriveTrigger(messages []json.RawMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		var p triggerPeek
		if json.Unmarshal(messages[i], &p) != nil {
			continue
		}
		if isSystemMessage(p) {
			continue
		}
		if isUserMessage(p) {
			return "user_message"
		}
		if isFunctionCallOutput(p) {
			return findToolName(messages, p.CallID, i-1)
		}
	}
	return ""
}

func cachedTokensFromUsage(usage *protocol.UsageResponse) int {
	if usage == nil || usage.InputTokensDetails == nil {
		return 0
	}
	return usage.InputTokensDetails.CachedTokens
}

func reasoningTokensFromUsage(usage *protocol.UsageResponse) int {
	if usage == nil || usage.OutputTokensDetails == nil {
		return 0
	}
	return usage.OutputTokensDetails.ReasoningTokens
}

func reasoningCost(reasoningTokens int, pricing prompts.Pricing) float64 {
	return float64(reasoningTokens) * pricing.Output / 1_000_000
}

func totalCost(inputCost, cachedCost, outputCost, rCost float64) float64 {
	return inputCost + cachedCost + outputCost + rCost
}

func BuildCallRecord(endpoint, model, reasoning, serviceTier, trigger string, usage *protocol.UsageResponse, pricing prompts.Pricing, durationMs int64) CallRecord {
	inputTokens := 0
	outputTokens := 0
	if usage != nil {
		inputTokens = usage.InputTokens
		outputTokens = usage.OutputTokens
	}
	cached := cachedTokensFromUsage(usage)
	rTokens := reasoningTokensFromUsage(usage)
	textOutputTokens := outputTokens - rTokens
	inputCost, cachedCost, outputCost := computeCosts(inputTokens, cached, textOutputTokens, pricing)
	rCost := reasoningCost(rTokens, pricing)
	return CallRecord{
		Endpoint:          endpoint,
		Model:             model,
		Reasoning:         reasoning,
		ServiceTier:       serviceTier,
		Trigger:           trigger,
		InputTokens:       inputTokens,
		CachedInputTokens: cached,
		OutputTokens:      textOutputTokens,
		ReasoningTokens:   rTokens,
		InputCost:         inputCost,
		CachedInputCost:   cachedCost,
		OutputCost:        outputCost,
		ReasoningCost:     rCost,
		TotalCost:         totalCost(inputCost, cachedCost, outputCost, rCost),
		DurationMs:        durationMs,
	}
}

func BuildEmbeddingCallRecord(model string, totalTokens, inputCount int, pricing prompts.Pricing, durationMs int64) CallRecord {
	inputCost, _, _ := computeCosts(totalTokens, 0, 0, pricing)
	return CallRecord{
		Endpoint:    "embeddings",
		Model:       model,
		InputTokens: totalTokens,
		InputCost:   inputCost,
		TotalCost:   inputCost,
		DurationMs:  durationMs,
		InputCount:  inputCount,
	}
}

func callRecordAttrs(rec CallRecord) []any {
	attrs := []any{
		slog.String("endpoint", rec.Endpoint),
		slog.String("model", rec.Model),
		slog.String("reasoning", rec.Reasoning),
		slog.String("service_tier", rec.ServiceTier),
		slog.String("trigger", rec.Trigger),
		slog.Int("input_tokens", rec.InputTokens),
		slog.Int("cached_input_tokens", rec.CachedInputTokens),
		slog.Int("output_tokens", rec.OutputTokens),
		slog.Int("reasoning_tokens", rec.ReasoningTokens),
		slog.Float64("input_cost", rec.InputCost),
		slog.Float64("cached_input_cost", rec.CachedInputCost),
		slog.Float64("output_cost", rec.OutputCost),
		slog.Float64("reasoning_cost", rec.ReasoningCost),
		slog.Float64("total_cost", rec.TotalCost),
		slog.Int64("duration_ms", rec.DurationMs),
	}
	if rec.InputCount > 0 {
		attrs = append(attrs, slog.Int("input_count", rec.InputCount))
	}
	return attrs
}

func LogCallRecord(ctx context.Context, rec CallRecord) {
	slog.InfoContext(ctx, "call completed",
		"component", "usage",
		slog.Group("data", callRecordAttrs(rec)...),
	)
	if axiomToken != "" && axiomDataset != "" {
		go sendToAxiom(ctx, rec)
	}
}
