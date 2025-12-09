package http

import "testing"

func TestShouldEnablePromptCaching(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"anthropic/claude-haiku-4.5", true},
		{"anthropic/claude-sonnet-3.5", true},
		{"claude-3-opus", true},
		{"deepseek/deepseek-v3.2", false},
		{"openai/gpt-4", false},
		{"meta-llama/llama-3", false},
		{"anthropic-claude-haiku", true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := shouldEnablePromptCaching(tt.model)
			if got != tt.want {
				t.Errorf("shouldEnablePromptCaching(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
