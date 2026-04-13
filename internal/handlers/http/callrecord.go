package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"hermes-logos/internal/logging"
	"hermes-logos/internal/prompts"
)

var axiomToken = "xaat-f8e2e9bf-7dd2-4237-8293-890f799b2101"
var axiomDataset = "mummu"

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

func deriveTrigger(messages []json.RawMessage) string {
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

func cachedTokensFromUsage(usage *UsageResponse) int {
	if usage == nil || usage.InputTokensDetails == nil {
		return 0
	}
	return usage.InputTokensDetails.CachedTokens
}

func reasoningTokensFromUsage(usage *UsageResponse) int {
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

func buildCallRecord(endpoint, model, reasoning, serviceTier, trigger string, usage *UsageResponse, pricing prompts.Pricing, durationMs int64) CallRecord {
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

func buildEmbeddingCallRecord(model string, totalTokens, inputCount int, pricing prompts.Pricing, durationMs int64) CallRecord {
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

func logCallRecord(ctx context.Context, rec CallRecord) {
	slog.InfoContext(ctx, "call completed",
		"component", "usage",
		slog.Group("data",
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
			slog.Int("input_count", rec.InputCount),
		),
	)
	if axiomToken != "" && axiomDataset != "" {
		go sendToAxiom(ctx, rec)
	}
}

type axiomEvent struct {
	Time              string  `json:"_time"`
	Endpoint          string  `json:"endpoint"`
	Model             string  `json:"model"`
	Reasoning         string  `json:"reasoning,omitempty"`
	ServiceTier       string  `json:"service_tier,omitempty"`
	Trigger           string  `json:"trigger,omitempty"`
	InputTokens       int     `json:"input_tokens"`
	CachedInputTokens int     `json:"cached_input_tokens"`
	OutputTokens      int     `json:"output_tokens"`
	ReasoningTokens   int     `json:"reasoning_tokens"`
	InputCost         float64 `json:"input_cost"`
	CachedInputCost   float64 `json:"cached_input_cost"`
	OutputCost        float64 `json:"output_cost"`
	ReasoningCost     float64 `json:"reasoning_cost"`
	TotalCost         float64 `json:"total_cost"`
	DurationMs        int64   `json:"duration_ms"`
	InputCount        int     `json:"input_count,omitempty"`
	RequestID         string  `json:"request_id,omitempty"`
	SessionID         string  `json:"session_id,omitempty"`
	ProjectID         string  `json:"project_id,omitempty"`
}

func buildAxiomEvent(ctx context.Context, rec CallRecord) axiomEvent {
	ev := axiomEvent{
		Time:              time.Now().UTC().Format(time.RFC3339Nano),
		Endpoint:          rec.Endpoint,
		Model:             rec.Model,
		Reasoning:         rec.Reasoning,
		ServiceTier:       rec.ServiceTier,
		Trigger:           rec.Trigger,
		InputTokens:       rec.InputTokens,
		CachedInputTokens: rec.CachedInputTokens,
		OutputTokens:      rec.OutputTokens,
		ReasoningTokens:   rec.ReasoningTokens,
		InputCost:         rec.InputCost,
		CachedInputCost:   rec.CachedInputCost,
		OutputCost:        rec.OutputCost,
		ReasoningCost:     rec.ReasoningCost,
		TotalCost:         rec.TotalCost,
		DurationMs:        rec.DurationMs,
		InputCount:        rec.InputCount,
	}
	for _, attr := range logging.AttrsFromContext(ctx) {
		switch attr.Key {
		case "request_id":
			ev.RequestID = attr.Value.String()
		case "session_id":
			ev.SessionID = attr.Value.String()
		case "project_id":
			ev.ProjectID = attr.Value.String()
		}
	}
	return ev
}

func sendToAxiom(ctx context.Context, rec CallRecord) {
	ev := buildAxiomEvent(ctx, rec)
	body, err := json.Marshal([]axiomEvent{ev})
	if err != nil {
		slog.Error("failed to marshal axiom event", "component", "usage", "error", err)
		return
	}

	url := "https://api.axiom.co/v1/datasets/" + axiomDataset + "/ingest"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		slog.Error("failed to create axiom request", "component", "usage", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+axiomToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to send to axiom", "component", "usage", "error", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("axiom ingest returned error", "component", "usage", slog.Group("data", slog.Int("status", resp.StatusCode)))
	}
}
