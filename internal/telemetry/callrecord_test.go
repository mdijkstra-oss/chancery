package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/mdijkstra-oss/chancery/internal/logging"
	"github.com/mdijkstra-oss/chancery/internal/protocol"

	"github.com/google/go-cmp/cmp"
)

type usageLogData struct {
	Endpoint          string `json:"endpoint"`
	Model             string `json:"model"`
	Reasoning         string `json:"reasoning"`
	ServiceTier       string `json:"service_tier"`
	Trigger           string `json:"trigger"`
	InputTokens       int    `json:"input_tokens"`
	CachedInputTokens int    `json:"cached_input_tokens"`
	CacheWriteTokens  int    `json:"cache_write_tokens"`
	OutputTokens      int    `json:"output_tokens"`
	ReasoningTokens   int    `json:"reasoning_tokens"`
	DurationMs        int64  `json:"duration_ms"`
	InputCount        int    `json:"input_count"`
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

func TestDeriveTrigger(t *testing.T) {
	cases := []struct {
		name     string
		messages []json.RawMessage
		want     string
	}{
		{name: "empty"},
		{name: "user message last", messages: []json.RawMessage{json.RawMessage(`{"role":"assistant","content":"hello"}`), json.RawMessage(`{"role":"user","content":"hi"}`)}, want: "user_message"},
		{name: "system skipped", messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi"}`), json.RawMessage(`{"role":"system","content":"planning"}`)}, want: "user_message"},
		{name: "tool output", messages: []json.RawMessage{json.RawMessage(`{"type":"function_call","call_id":"c1","name":"shell"}`), json.RawMessage(`{"type":"function_call_output","call_id":"c1"}`)}, want: "shell"},
		{name: "latest tool output", messages: []json.RawMessage{json.RawMessage(`{"type":"function_call","call_id":"c1","name":"search"}`), json.RawMessage(`{"type":"function_call_output","call_id":"c1"}`), json.RawMessage(`{"type":"function_call","call_id":"c2","name":"shell"}`), json.RawMessage(`{"type":"function_call_output","call_id":"c2"}`)}, want: "shell"},
		{name: "reasoning skipped", messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi"}`), json.RawMessage(`{"type":"reasoning"}`)}, want: "user_message"},
		{name: "system only", messages: []json.RawMessage{json.RawMessage(`{"role":"system","content":"help"}`)}},
		{name: "orphan tool output", messages: []json.RawMessage{json.RawMessage(`{"type":"function_call_output","call_id":"missing"}`)}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := DeriveTrigger(test.messages); got != test.want {
				t.Errorf("DeriveTrigger() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBuildCallRecord(t *testing.T) {
	tests := []struct {
		name  string
		usage *protocol.UsageResponse
		want  CallRecord
	}{
		{
			name:  "plain usage",
			usage: &protocol.UsageResponse{InputTokens: 10000, OutputTokens: 2000, TotalTokens: 12000, InputTokensDetails: &protocol.PromptTokensDetails{CachedTokens: 6000}},
			want:  CallRecord{Endpoint: "agent", Model: "model", Reasoning: "low", ServiceTier: "priority", Trigger: "shell", InputTokens: 10000, CachedInputTokens: 6000, OutputTokens: 2000, DurationMs: 1234},
		},
		{
			name:  "reasoning and cache write",
			usage: &protocol.UsageResponse{InputTokens: 10000, OutputTokens: 5000, TotalTokens: 15000, InputTokensDetails: &protocol.PromptTokensDetails{CachedTokens: 2000, CacheCreationTokens: 3000}, OutputTokensDetails: &protocol.OutputTokensDetails{ReasoningTokens: 3000}},
			want:  CallRecord{Endpoint: "agent", Model: "model", Reasoning: "low", ServiceTier: "priority", Trigger: "shell", InputTokens: 10000, CachedInputTokens: 2000, CacheWriteTokens: 3000, OutputTokens: 2000, ReasoningTokens: 3000, DurationMs: 1234},
		},
		{
			name: "nil usage",
			want: CallRecord{Endpoint: "agent", Model: "model", Reasoning: "low", ServiceTier: "priority", Trigger: "shell", DurationMs: 1234},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := BuildCallRecord("agent", "model", "low", "priority", "shell", test.usage, 1234)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("record mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildEmbeddingCallRecord(t *testing.T) {
	want := CallRecord{Endpoint: "embeddings", Model: "embedding-model", Trigger: "embeddings", InputTokens: 5000, DurationMs: 300, InputCount: 10}
	got := BuildEmbeddingCallRecord("embedding-model", 5000, 10, 300)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("record mismatch (-want +got):\n%s", diff)
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
	record := CallRecord{
		Endpoint:          "agent",
		Model:             "model",
		Reasoning:         "high",
		ServiceTier:       "priority",
		Trigger:           "shell",
		InputTokens:       10000,
		CachedInputTokens: 6000,
		CacheWriteTokens:  1000,
		OutputTokens:      2000,
		ReasoningTokens:   500,
		DurationMs:        1234,
		InputCount:        4,
	}
	LogCallRecord(ctx, record)

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
			Endpoint:          record.Endpoint,
			Model:             record.Model,
			Reasoning:         record.Reasoning,
			ServiceTier:       record.ServiceTier,
			Trigger:           record.Trigger,
			InputTokens:       record.InputTokens,
			CachedInputTokens: record.CachedInputTokens,
			CacheWriteTokens:  record.CacheWriteTokens,
			OutputTokens:      record.OutputTokens,
			ReasoningTokens:   record.ReasoningTokens,
			DurationMs:        record.DurationMs,
			InputCount:        record.InputCount,
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("usage log mismatch (-want +got):\n%s", diff)
	}
}
