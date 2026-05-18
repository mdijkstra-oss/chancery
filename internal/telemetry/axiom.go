package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"hermes-logos/internal/logging"
)

var axiomToken = "xaat-f8e2e9bf-7dd2-4237-8293-890f799b2101"
var axiomDataset = "mummu"

type axiomEvent struct {
	Time              string  `json:"_time"`
	Endpoint          string  `json:"endpoint"`
	Model             string  `json:"model"`
	Reasoning         string  `json:"reasoning,omitempty"`
	ServiceTier       string  `json:"service_tier,omitempty"`
	Trigger           string  `json:"trigger,omitempty"`
	InputTokens       int     `json:"input_tokens"`
	CachedInputTokens int     `json:"cached_input_tokens"`
	CacheWriteTokens  int     `json:"cache_write_tokens"`
	OutputTokens      int     `json:"output_tokens"`
	ReasoningTokens   int     `json:"reasoning_tokens"`
	InputCost         float64 `json:"input_cost"`
	CachedInputCost   float64 `json:"cached_input_cost"`
	CacheWriteCost    float64 `json:"cache_write_cost"`
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
		CacheWriteTokens:  rec.CacheWriteTokens,
		OutputTokens:      rec.OutputTokens,
		ReasoningTokens:   rec.ReasoningTokens,
		InputCost:         rec.InputCost,
		CachedInputCost:   rec.CachedInputCost,
		CacheWriteCost:    rec.CacheWriteCost,
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
