package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"testing"

	"hermes-logos/internal/logging"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"

	"github.com/google/go-cmp/cmp"
)

type usageLogData struct {
	Endpoint          string  `json:"endpoint"`
	Model             string  `json:"model"`
	Reasoning         string  `json:"reasoning"`
	ServiceTier       string  `json:"service_tier"`
	Trigger           string  `json:"trigger"`
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
	InputCount        int     `json:"input_count"`
}

type usageLogRecord struct {
	Message     string            `json:"msg"`
	Environment string            `json:"environment"`
	Component   string            `json:"component"`
	RequestID   string            `json:"request_id"`
	User        string            `json:"user"`
	Headers     map[string]string `json:"headers"`
	Data        usageLogData      `json:"data"`
}

func TestComputeCosts(t *testing.T) {
	cases := []struct {
		name                       string
		input, cached, creation    int
		output                     int
		pricing                    prompts.Pricing
		wantInput, wantCached      float64
		wantCacheWrite, wantOutput float64
	}{
		{
			"zero tokens",
			0, 0, 0, 0,
			prompts.Pricing{Input: 1.75, Output: 14.00, CachedInput: 0.18},
			0, 0, 0, 0,
		},
		{
			"all uncached",
			1000, 0, 0, 500,
			prompts.Pricing{Input: 1.75, Output: 14.00, CachedInput: 0.18},
			0.00175, 0, 0, 0.007,
		},
		{
			"all cached",
			1000, 1000, 0, 500,
			prompts.Pricing{Input: 1.75, Output: 14.00, CachedInput: 0.18},
			0, 0.00018, 0, 0.007,
		},
		{
			"mixed cached and uncached",
			10000, 6000, 0, 2000,
			prompts.Pricing{Input: 1.75, Output: 14.00, CachedInput: 0.18},
			0.007, 0.00108, 0, 0.028,
		},
		{
			"embedding pricing",
			5000, 0, 0, 0,
			prompts.Pricing{Input: 0.13, Output: 0, CachedInput: 0},
			0.00065, 0, 0, 0,
		},
		{
			"cache write tokens",
			10000, 2000, 5000, 1000,
			prompts.Pricing{Input: 3.00, Output: 15.00, CachedInput: 0.30, CacheWriteInput: 3.75},
			0.009, 0.0006, 0.01875, 0.015,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotInput, gotCached, gotCacheWrite, gotOutput := computeCosts(tc.input, tc.cached, tc.creation, tc.output, tc.pricing)
			if !approxEqual(gotInput, tc.wantInput) {
				t.Errorf("inputCost = %f, want %f", gotInput, tc.wantInput)
			}
			if !approxEqual(gotCached, tc.wantCached) {
				t.Errorf("cachedCost = %f, want %f", gotCached, tc.wantCached)
			}
			if !approxEqual(gotCacheWrite, tc.wantCacheWrite) {
				t.Errorf("cacheWriteCost = %f, want %f", gotCacheWrite, tc.wantCacheWrite)
			}
			if !approxEqual(gotOutput, tc.wantOutput) {
				t.Errorf("outputCost = %f, want %f", gotOutput, tc.wantOutput)
			}
		})
	}
}

func TestDeriveTrigger(t *testing.T) {
	cases := []struct {
		name     string
		messages []json.RawMessage
		want     string
	}{
		{
			"empty messages",
			nil,
			"",
		},
		{
			"user message last",
			[]json.RawMessage{
				json.RawMessage(`{"role":"assistant","content":"hello"}`),
				json.RawMessage(`{"role":"user","content":"hi"}`),
			},
			"user_message",
		},
		{
			"system message skipped to find user",
			[]json.RawMessage{
				json.RawMessage(`{"role":"user","content":"hi"}`),
				json.RawMessage(`{"role":"system","content":"<!-- prompt: planning -->"}`),
			},
			"user_message",
		},
		{
			"function call output finds tool name",
			[]json.RawMessage{
				json.RawMessage(`{"role":"user","content":"do it"}`),
				json.RawMessage(`{"type":"function_call","call_id":"c1","name":"shell","arguments":"{}"}`),
				json.RawMessage(`{"type":"function_call_output","call_id":"c1","output":"ok"}`),
			},
			"shell",
		},
		{
			"function call output with multiple tools finds correct one",
			[]json.RawMessage{
				json.RawMessage(`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`),
				json.RawMessage(`{"type":"function_call_output","call_id":"c1","output":"results"}`),
				json.RawMessage(`{"type":"function_call","call_id":"c2","name":"shell","arguments":"{}"}`),
				json.RawMessage(`{"type":"function_call_output","call_id":"c2","output":"done"}`),
			},
			"shell",
		},
		{
			"skips reasoning to find user message",
			[]json.RawMessage{
				json.RawMessage(`{"role":"user","content":"hi"}`),
				json.RawMessage(`{"type":"reasoning","id":"r1"}`),
			},
			"user_message",
		},
		{
			"system only returns empty",
			[]json.RawMessage{
				json.RawMessage(`{"role":"system","content":"you are helpful"}`),
			},
			"",
		},
		{
			"function call output with no matching call",
			[]json.RawMessage{
				json.RawMessage(`{"type":"function_call_output","call_id":"c99","output":"orphan"}`),
			},
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DeriveTrigger(tc.messages)
			if got != tc.want {
				t.Errorf("DeriveTrigger() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildCallRecord(t *testing.T) {
	pricing := prompts.Pricing{Input: 1.75, Output: 14.00, CachedInput: 0.18}
	usage := &protocol.UsageResponse{
		InputTokens:  10000,
		OutputTokens: 2000,
		TotalTokens:  12000,
		InputTokensDetails: &protocol.PromptTokensDetails{
			CachedTokens: 6000,
		},
	}

	rec := BuildCallRecord("qual-coder", "gpt-5.2", "low", "priority", "shell", usage, pricing, 1234)

	if rec.Endpoint != "qual-coder" {
		t.Errorf("Endpoint = %q, want %q", rec.Endpoint, "qual-coder")
	}
	if rec.InputTokens != 10000 {
		t.Errorf("InputTokens = %d, want %d", rec.InputTokens, 10000)
	}
	if rec.CachedInputTokens != 6000 {
		t.Errorf("CachedInputTokens = %d, want %d", rec.CachedInputTokens, 6000)
	}
	if rec.OutputTokens != 2000 {
		t.Errorf("OutputTokens = %d, want %d", rec.OutputTokens, 2000)
	}
	if rec.DurationMs != 1234 {
		t.Errorf("DurationMs = %d, want %d", rec.DurationMs, 1234)
	}
	if rec.Trigger != "shell" {
		t.Errorf("Trigger = %q, want %q", rec.Trigger, "shell")
	}
	if !approxEqual(rec.InputCost, 0.007) {
		t.Errorf("InputCost = %f, want %f", rec.InputCost, 0.007)
	}
	if !approxEqual(rec.CachedInputCost, 0.00108) {
		t.Errorf("CachedInputCost = %f, want %f", rec.CachedInputCost, 0.00108)
	}
	if !approxEqual(rec.OutputCost, 0.028) {
		t.Errorf("OutputCost = %f, want %f", rec.OutputCost, 0.028)
	}
	if rec.ReasoningTokens != 0 {
		t.Errorf("ReasoningTokens = %d, want 0", rec.ReasoningTokens)
	}
	if rec.ReasoningCost != 0 {
		t.Errorf("ReasoningCost = %f, want 0", rec.ReasoningCost)
	}
}

func TestBuildCallRecordWithReasoning(t *testing.T) {
	pricing := prompts.Pricing{Input: 1.75, Output: 14.00, CachedInput: 0.18}
	usage := &protocol.UsageResponse{
		InputTokens:  10000,
		OutputTokens: 5000,
		TotalTokens:  15000,
		InputTokensDetails: &protocol.PromptTokensDetails{
			CachedTokens: 4000,
		},
		OutputTokensDetails: &protocol.OutputTokensDetails{
			ReasoningTokens: 3000,
		},
	}

	rec := BuildCallRecord("qual-coder", "gpt-5.2", "low", "priority", "user_message", usage, pricing, 2000)

	if rec.ReasoningTokens != 3000 {
		t.Errorf("ReasoningTokens = %d, want %d", rec.ReasoningTokens, 3000)
	}
	wantReasoningCost := 3000.0 * 14.00 / 1_000_000
	if !approxEqual(rec.ReasoningCost, wantReasoningCost) {
		t.Errorf("ReasoningCost = %f, want %f", rec.ReasoningCost, wantReasoningCost)
	}
	if rec.OutputTokens != 2000 {
		t.Errorf("OutputTokens = %d, want %d (5000 total - 3000 reasoning)", rec.OutputTokens, 2000)
	}
	wantOutputCost := 2000.0 * 14.00 / 1_000_000
	if !approxEqual(rec.OutputCost, wantOutputCost) {
		t.Errorf("OutputCost = %f, want %f", rec.OutputCost, wantOutputCost)
	}
}

func TestBuildCallRecordWithCacheWrite(t *testing.T) {
	pricing := prompts.Pricing{Input: 3.00, Output: 15.00, CachedInput: 0.30, CacheWriteInput: 3.75}
	usage := &protocol.UsageResponse{
		InputTokens:  10000,
		OutputTokens: 1000,
		TotalTokens:  11000,
		InputTokensDetails: &protocol.PromptTokensDetails{
			CachedTokens:        2000,
			CacheCreationTokens: 5000,
		},
	}

	rec := BuildCallRecord("qual-coder", "claude-sonnet-4-6", "minimal", "", "user_message", usage, pricing, 3000)

	if rec.CacheWriteTokens != 5000 {
		t.Errorf("CacheWriteTokens = %d, want %d", rec.CacheWriteTokens, 5000)
	}
	wantCacheWriteCost := 5000.0 * 3.75 / 1_000_000
	if !approxEqual(rec.CacheWriteCost, wantCacheWriteCost) {
		t.Errorf("CacheWriteCost = %f, want %f", rec.CacheWriteCost, wantCacheWriteCost)
	}
	wantInputCost := 3000.0 * 3.00 / 1_000_000
	if !approxEqual(rec.InputCost, wantInputCost) {
		t.Errorf("InputCost = %f, want %f (uncached: 10000-2000-5000=3000)", rec.InputCost, wantInputCost)
	}
}

func TestBuildCallRecordNilUsage(t *testing.T) {
	pricing := prompts.Pricing{Input: 0.25, Output: 2.00, CachedInput: 0.03}
	rec := BuildCallRecord("compacter", "gpt-5-mini", "", "", "user_message", nil, pricing, 500)

	if rec.InputTokens != 0 {
		t.Errorf("InputTokens = %d, want 0", rec.InputTokens)
	}
	if rec.OutputTokens != 0 {
		t.Errorf("OutputTokens = %d, want 0", rec.OutputTokens)
	}
}

func TestBuildEmbeddingCallRecord(t *testing.T) {
	pricing := prompts.Pricing{Input: 0.13, Output: 0, CachedInput: 0}
	rec := BuildEmbeddingCallRecord("text-embedding-3-large", 5000, 10, pricing, 300)

	if rec.Endpoint != "embeddings" {
		t.Errorf("Endpoint = %q, want %q", rec.Endpoint, "embeddings")
	}
	if rec.InputTokens != 5000 {
		t.Errorf("InputTokens = %d, want %d", rec.InputTokens, 5000)
	}
	if rec.OutputTokens != 0 {
		t.Errorf("OutputTokens = %d, want 0", rec.OutputTokens)
	}
	if rec.InputCount != 10 {
		t.Errorf("InputCount = %d, want %d", rec.InputCount, 10)
	}
	if !approxEqual(rec.InputCost, 0.00065) {
		t.Errorf("InputCost = %f, want %f", rec.InputCost, 0.00065)
	}
}

func TestLogCallRecord(t *testing.T) {
	var output bytes.Buffer
	originalLogger := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	handler := logging.NewContextHandler(slog.NewJSONHandler(&output, nil))
	slog.SetDefault(slog.New(handler).With("environment", "test"))

	ctx := logging.WithAttr(context.Background(), "request_id", "req-123")
	ctx = logging.WithAttr(ctx, "user", "user-012")
	ctx = logging.WithAttrs(ctx, slog.Group("headers", slog.String("x-session-id", "sess-456"), slog.String("x-project-id", "proj-789")))
	rec := CallRecord{
		Endpoint:          "qual-coder",
		Model:             "gpt-5",
		Reasoning:         "high",
		ServiceTier:       "priority",
		Trigger:           "shell",
		InputTokens:       10000,
		CachedInputTokens: 6000,
		CacheWriteTokens:  1000,
		OutputTokens:      2000,
		ReasoningTokens:   500,
		InputCost:         0.007,
		CachedInputCost:   0.001,
		CacheWriteCost:    0.002,
		OutputCost:        0.028,
		ReasoningCost:     0.007,
		TotalCost:         0.045,
		DurationMs:        1234,
		InputCount:        4,
	}

	LogCallRecord(ctx, rec)

	var got usageLogRecord
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal usage log: %v", err)
	}
	want := usageLogRecord{
		Message:     "call completed",
		Environment: "test",
		Component:   "usage",
		RequestID:   "req-123",
		User:        "user-012",
		Headers:     map[string]string{"x-session-id": "sess-456", "x-project-id": "proj-789"},
		Data: usageLogData{
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
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("usage log mismatch (-want +got):\n%s", diff)
	}
}

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
