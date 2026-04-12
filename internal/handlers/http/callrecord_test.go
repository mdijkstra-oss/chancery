package http

import (
	"encoding/json"
	"math"
	"testing"

	"hermes-logos/internal/prompts"
)

func TestComputeCosts(t *testing.T) {
	cases := []struct {
		name                                     string
		input, cached, output                    int
		pricing                                  prompts.Pricing
		wantInput, wantCached, wantOutput        float64
	}{
		{
			"zero tokens",
			0, 0, 0,
			prompts.Pricing{Input: 175, Output: 1400, CachedInput: 17.5},
			0, 0, 0,
		},
		{
			"all uncached",
			1000, 0, 500,
			prompts.Pricing{Input: 175, Output: 1400, CachedInput: 17.5},
			0.175, 0, 0.7,
		},
		{
			"all cached",
			1000, 1000, 500,
			prompts.Pricing{Input: 175, Output: 1400, CachedInput: 17.5},
			0, 0.0175, 0.7,
		},
		{
			"mixed cached and uncached",
			10000, 6000, 2000,
			prompts.Pricing{Input: 175, Output: 1400, CachedInput: 17.5},
			0.7, 0.105, 2.8,
		},
		{
			"embedding pricing",
			5000, 0, 0,
			prompts.Pricing{Input: 0.13, Output: 0, CachedInput: 0},
			0.00065, 0, 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotInput, gotCached, gotOutput := computeCosts(tc.input, tc.cached, tc.output, tc.pricing)
			if !approxEqual(gotInput, tc.wantInput) {
				t.Errorf("inputCost = %f, want %f", gotInput, tc.wantInput)
			}
			if !approxEqual(gotCached, tc.wantCached) {
				t.Errorf("cachedCost = %f, want %f", gotCached, tc.wantCached)
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
			got := deriveTrigger(tc.messages)
			if got != tc.want {
				t.Errorf("deriveTrigger() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildCallRecord(t *testing.T) {
	pricing := prompts.Pricing{Input: 175, Output: 1400, CachedInput: 17.5}
	usage := &UsageResponse{
		InputTokens:  10000,
		OutputTokens: 2000,
		TotalTokens:  12000,
		InputTokensDetails: &PromptTokensDetails{
			CachedTokens: 6000,
		},
	}

	rec := buildCallRecord("qual-coder", "gpt-5.4", "low", "priority", "shell", usage, pricing, 1234)

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
	if !approxEqual(rec.InputCost, 0.7) {
		t.Errorf("InputCost = %f, want %f", rec.InputCost, 0.7)
	}
	if !approxEqual(rec.CachedInputCost, 0.105) {
		t.Errorf("CachedInputCost = %f, want %f", rec.CachedInputCost, 0.105)
	}
	if !approxEqual(rec.OutputCost, 2.8) {
		t.Errorf("OutputCost = %f, want %f", rec.OutputCost, 2.8)
	}
}

func TestBuildCallRecordNilUsage(t *testing.T) {
	pricing := prompts.Pricing{Input: 175, Output: 1400, CachedInput: 17.5}
	rec := buildCallRecord("compacter", "gpt-5-mini", "", "", "user_message", nil, pricing, 500)

	if rec.InputTokens != 0 {
		t.Errorf("InputTokens = %d, want 0", rec.InputTokens)
	}
	if rec.OutputTokens != 0 {
		t.Errorf("OutputTokens = %d, want 0", rec.OutputTokens)
	}
}

func TestBuildEmbeddingCallRecord(t *testing.T) {
	pricing := prompts.Pricing{Input: 0.13, Output: 0, CachedInput: 0}
	rec := buildEmbeddingCallRecord("text-embedding-3-large", 5000, 10, pricing, 300)

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

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
