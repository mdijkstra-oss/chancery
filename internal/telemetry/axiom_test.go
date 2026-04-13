package telemetry

import (
	"context"
	"testing"

	"hermes-logos/internal/logging"
)

func TestBuildAxiomEvent(t *testing.T) {
	rec := CallRecord{
		Endpoint:          "qual-coder",
		Model:             "gpt-5",
		Reasoning:         "high",
		ServiceTier:       "flex",
		Trigger:           "shell",
		InputTokens:       10000,
		CachedInputTokens: 6000,
		OutputTokens:      2000,
		ReasoningTokens:   500,
		InputCost:         0.007,
		CachedInputCost:   0.00108,
		OutputCost:        0.028,
		ReasoningCost:     0.007,
		TotalCost:         0.04308,
		DurationMs:        1234,
		InputCount:        0,
	}

	ctx := context.Background()
	ctx = logging.WithAttr(ctx, "request_id", "req-123")
	ctx = logging.WithAttr(ctx, "session_id", "sess-456")
	ctx = logging.WithAttr(ctx, "project_id", "proj-789")

	ev := buildAxiomEvent(ctx, rec)

	cases := []struct {
		name string
		got  any
		want any
	}{
		{"endpoint", ev.Endpoint, "qual-coder"},
		{"model", ev.Model, "gpt-5"},
		{"reasoning", ev.Reasoning, "high"},
		{"service_tier", ev.ServiceTier, "flex"},
		{"trigger", ev.Trigger, "shell"},
		{"input_tokens", ev.InputTokens, 10000},
		{"cached_input_tokens", ev.CachedInputTokens, 6000},
		{"output_tokens", ev.OutputTokens, 2000},
		{"reasoning_tokens", ev.ReasoningTokens, 500},
		{"duration_ms", ev.DurationMs, int64(1234)},
		{"request_id", ev.RequestID, "req-123"},
		{"session_id", ev.SessionID, "sess-456"},
		{"project_id", ev.ProjectID, "proj-789"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %v, want %v", tc.got, tc.want)
			}
		})
	}

	if ev.Time == "" {
		t.Error("time should not be empty")
	}
}

func TestBuildAxiomEventEmptyContext(t *testing.T) {
	rec := CallRecord{Endpoint: "test", Model: "m"}
	ev := buildAxiomEvent(context.Background(), rec)

	if ev.RequestID != "" {
		t.Errorf("request_id = %q, want empty", ev.RequestID)
	}
	if ev.SessionID != "" {
		t.Errorf("session_id = %q, want empty", ev.SessionID)
	}
	if ev.ProjectID != "" {
		t.Errorf("project_id = %q, want empty", ev.ProjectID)
	}
}
