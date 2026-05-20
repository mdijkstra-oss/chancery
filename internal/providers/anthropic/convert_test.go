package anthropic

import (
	"encoding/json"
	"testing"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
)

func TestMessagesToAnthropic(t *testing.T) {
	cases := []struct {
		name            string
		messages        []json.RawMessage
		breakpoints     map[int]bool
		wantLen         int
		checkFirst      func(t *testing.T, msg Message)
	}{
		{
			"user message",
			[]json.RawMessage{
				json.RawMessage(`{"role":"user","content":"hello"}`),
			},
			nil,
			1,
			func(t *testing.T, msg Message) {
				if msg.Role != "user" {
					t.Errorf("role = %q, want user", msg.Role)
				}
				if msg.Content[0].Text != "hello" {
					t.Errorf("text = %q, want hello", msg.Content[0].Text)
				}
			},
		},
		{
			"system becomes user with wrapper",
			[]json.RawMessage{
				json.RawMessage(`{"role":"system","content":"be helpful"}`),
			},
			nil,
			1,
			func(t *testing.T, msg Message) {
				if msg.Role != "user" {
					t.Errorf("role = %q, want user", msg.Role)
				}
				if msg.Content[0].Text != "<system_message>\nbe helpful\n</system_message>" {
					t.Errorf("text = %q", msg.Content[0].Text)
				}
			},
		},
		{
			"consecutive same role merged",
			[]json.RawMessage{
				json.RawMessage(`{"role":"user","content":"one"}`),
				json.RawMessage(`{"role":"user","content":"two"}`),
			},
			nil,
			1,
			func(t *testing.T, msg Message) {
				if len(msg.Content) != 2 {
					t.Errorf("content blocks = %d, want 2", len(msg.Content))
				}
			},
		},
		{
			"function call and output",
			[]json.RawMessage{
				json.RawMessage(`{"type":"function_call","call_id":"c1","name":"search","arguments":"{\"q\":\"test\"}"}`),
				json.RawMessage(`{"type":"function_call_output","call_id":"c1","output":"results"}`),
			},
			nil,
			2,
			func(t *testing.T, msg Message) {
				if msg.Role != "assistant" {
					t.Errorf("role = %q, want assistant", msg.Role)
				}
				if msg.Content[0].Type != "tool_use" {
					t.Errorf("type = %q, want tool_use", msg.Content[0].Type)
				}
				if msg.Content[0].Name != "search" {
					t.Errorf("name = %q, want search", msg.Content[0].Name)
				}
			},
		},
		{
			"breakpoint adds cache_control",
			[]json.RawMessage{
				json.RawMessage(`{"role":"user","content":"cached content"}`),
				json.RawMessage(`{"role":"assistant","content":"reply"}`),
			},
			map[int]bool{0: true},
			2,
			func(t *testing.T, msg Message) {
				if msg.Content[0].CacheControl == nil {
					t.Error("expected cache_control on breakpoint message")
				}
				if msg.Content[0].CacheControl.Type != "ephemeral" {
					t.Errorf("cache_control.type = %q, want ephemeral", msg.Content[0].CacheControl.Type)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := MessagesToAnthropic(tc.messages, tc.breakpoints)
			if len(result) != tc.wantLen {
				t.Fatalf("message count = %d, want %d", len(result), tc.wantLen)
			}
			if tc.checkFirst != nil {
				tc.checkFirst(t, result[0])
			}
		})
	}
}

func TestToolsToAnthropic(t *testing.T) {
	cases := []struct {
		name    string
		tools   []json.RawMessage
		wantLen int
	}{
		{
			"empty tools",
			nil,
			0,
		},
		{
			"single tool",
			[]json.RawMessage{
				json.RawMessage(`{"name":"search","description":"search docs","parameters":{"type":"object","properties":{"q":{"type":"string"}}}}`),
			},
			1,
		},
		{
			"invalid tool skipped",
			[]json.RawMessage{
				json.RawMessage(`{"invalid": true}`),
				json.RawMessage(`{"name":"search","parameters":{}}`),
			},
			1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ToolsToAnthropic(tc.tools)
			if len(result) != tc.wantLen {
				t.Errorf("tool count = %d, want %d", len(result), tc.wantLen)
			}
		})
	}
}

func TestBuildThinkingConfig(t *testing.T) {
	cases := []struct {
		name     string
		effort   string
		wantType string
	}{
		{"empty effort", "", "disabled"},
		{"off effort", "off", "disabled"},
		{"none effort", "none", "disabled"},
		{"minimal effort", "minimal", "adaptive"},
		{"low effort", "low", "adaptive"},
		{"medium effort", "medium", "adaptive"},
		{"high effort", "high", "adaptive"},
		{"max effort", "max", "adaptive"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := buildThinkingConfig(tc.effort)
			if cfg.Type != tc.wantType {
				t.Errorf("type = %q, want %q", cfg.Type, tc.wantType)
			}
		})
	}
}

func TestBuildOutputConfig(t *testing.T) {
	cases := []struct {
		name       string
		effort     string
		wantNil    bool
		wantEffort string
	}{
		{"empty effort", "", true, ""},
		{"off effort", "off", true, ""},
		{"none effort", "none", true, ""},
		{"minimal maps to low", "minimal", false, "low"},
		{"low effort", "low", false, "low"},
		{"medium effort", "medium", false, "medium"},
		{"high effort", "high", false, "high"},
		{"max effort", "max", false, "max"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := buildOutputConfig(tc.effort)
			if tc.wantNil {
				if cfg != nil {
					t.Errorf("got %+v, want nil", cfg)
				}
				return
			}
			if cfg == nil {
				t.Fatal("got nil, want non-nil")
			}
			if cfg.Effort != tc.wantEffort {
				t.Errorf("effort = %q, want %q", cfg.Effort, tc.wantEffort)
			}
		})
	}
}

func TestBuildToolChoice(t *testing.T) {
	cases := []struct {
		name   string
		choice string
		want   any
	}{
		{"empty", "", nil},
		{"required", "required", map[string]string{"type": "any"}},
		{"none", "none", map[string]string{"type": "none"}},
		{"specific", "search", map[string]string{"type": "tool", "name": "search"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildToolChoice(tc.choice)
			if tc.want == nil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("got %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestSystemToAnthropic(t *testing.T) {
	cases := []struct {
		name           string
		prompt         string
		hasBreakpoints bool
		wantLen        int
		wantCache      bool
	}{
		{"empty prompt", "", false, 0, false},
		{"no breakpoints", "system prompt", false, 1, false},
		{"with breakpoints", "system prompt", true, 1, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := SystemToAnthropic(tc.prompt, tc.hasBreakpoints)
			if len(result) != tc.wantLen {
				t.Fatalf("block count = %d, want %d", len(result), tc.wantLen)
			}
			if tc.wantLen > 0 {
				hasCache := result[0].CacheControl != nil
				if hasCache != tc.wantCache {
					t.Errorf("cache_control present = %v, want %v", hasCache, tc.wantCache)
				}
			}
		})
	}
}

func TestBuildRequest(t *testing.T) {
	params := protocol.RequestParams{
		Model:           "claude-sonnet-4-6",
		SystemPrompt:    "be helpful",
		ReasoningEffort: "minimal",
		MaxTokens:       16384,
		Messages: []json.RawMessage{
			json.RawMessage(`{"role":"user","content":"hello"}`),
		},
		Tools: []json.RawMessage{
			json.RawMessage(`{"name":"search","description":"search","parameters":{"type":"object"}}`),
		},
	}

	req := BuildRequest(params, defaultProvider())
	if req.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q", req.Model)
	}
	if req.MaxTokens != 16384 {
		t.Errorf("max_tokens = %d, want 16384", req.MaxTokens)
	}
	if !req.Stream {
		t.Error("stream should be true")
	}
	if len(req.Messages) != 1 {
		t.Errorf("message count = %d, want 1", len(req.Messages))
	}
	if len(req.Tools) != 1 {
		t.Errorf("tool count = %d, want 1", len(req.Tools))
	}
	if req.Thinking == nil || req.Thinking.Type != "adaptive" {
		t.Error("expected adaptive thinking for minimal effort")
	}
}

func TestBuildRequestExplicitBreakpoints(t *testing.T) {
	params := protocol.RequestParams{
		Model:            "claude-sonnet-4-6",
		SystemPrompt:     "be helpful",
		ReasoningEffort:  "minimal",
		MaxTokens:        16384,
		CacheBreakpoints: map[int]bool{0: true},
		Messages: []json.RawMessage{
			json.RawMessage(`{"role":"user","content":"first"}`),
			json.RawMessage(`{"role":"assistant","content":"reply"}`),
			json.RawMessage(`{"role":"user","content":"second"}`),
		},
	}

	req := BuildRequest(params, defaultProvider())
	if req.System[0].CacheControl == nil {
		t.Error("system should have cache_control when breakpoints exist")
	}
	first := req.Messages[0]
	if first.Content[0].CacheControl == nil {
		t.Error("breakpointed message should have cache_control")
	}
	penultimate := req.Messages[1]
	if penultimate.Content[0].CacheControl != nil {
		t.Error("non-breakpointed message should not have cache_control")
	}
}

func defaultProvider() prompts.ProviderConfig {
	return prompts.ProviderConfig{
		Protocol: "anthropic",
		BaseURL:  "https://api.anthropic.com",
		APIKey:   "test-key",
	}
}
