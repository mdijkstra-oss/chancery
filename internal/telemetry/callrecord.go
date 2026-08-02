package telemetry

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/mdijkstra-oss/chancery/internal/protocol"
)

type CallRecord struct {
	Endpoint          string
	Model             string
	Reasoning         string
	ServiceTier       string
	Trigger           string
	InputTokens       int
	CachedInputTokens int
	CacheWriteTokens  int
	OutputTokens      int
	ReasoningTokens   int
	DurationMs        int64
	InputCount        int
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

func BuildCallRecord(endpoint, model, reasoning, serviceTier, trigger string, usage *protocol.UsageResponse, durationMs int64) CallRecord {
	inputTokens := 0
	outputTokens := 0
	if usage != nil {
		inputTokens = usage.InputTokens
		outputTokens = usage.OutputTokens
	}
	cached := protocol.CachedInputTokens(usage)
	cacheCreation := protocol.CacheWriteTokens(usage)
	rTokens := protocol.ReasoningTokens(usage)
	textOutputTokens := outputTokens - rTokens
	return CallRecord{
		Endpoint:          endpoint,
		Model:             model,
		Reasoning:         reasoning,
		ServiceTier:       serviceTier,
		Trigger:           trigger,
		InputTokens:       inputTokens,
		CachedInputTokens: cached,
		CacheWriteTokens:  cacheCreation,
		OutputTokens:      textOutputTokens,
		ReasoningTokens:   rTokens,
		DurationMs:        durationMs,
	}
}

func BuildEmbeddingCallRecord(model string, totalTokens, inputCount int, durationMs int64) CallRecord {
	return CallRecord{
		Endpoint:    "embeddings",
		Model:       model,
		Trigger:     "embeddings",
		InputTokens: totalTokens,
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
		slog.Int("cache_write_tokens", rec.CacheWriteTokens),
		slog.Int("output_tokens", rec.OutputTokens),
		slog.Int("reasoning_tokens", rec.ReasoningTokens),
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
}
