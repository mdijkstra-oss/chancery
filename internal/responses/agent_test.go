package responses

import (
	"testing"

	"github.com/mdijkstra-oss/chancery/internal/prompts"
)

func TestAgentFrom(t *testing.T) {
	cases := []struct {
		name     string
		resolved prompts.ResolvedAgent
		want     Agent
	}{{
		name: "frontmatter fields reach their body positions",
		resolved: prompts.ResolvedAgent{
			Path:   "hound",
			Prompt: prompts.CompiledAgent{Prompt: "You are a hound."},
			Config: prompts.PromptConfig{
				Model:            "openai/gpt-5.5",
				ReasoningEffort:  "high",
				ReasoningSummary: "auto",
				Verbosity:        "low",
				ServiceTier:      "priority",
				MaxTokens:        4096,
			},
		},
		want: Agent{
			Model:            "openai/gpt-5.5",
			Instructions:     "You are a hound.",
			ReasoningEffort:  "high",
			ReasoningSummary: "auto",
			Verbosity:        "low",
			ServiceTier:      "priority",
			MaxOutputTokens:  4096,
		},
	}, {
		name: "a model's own prompt goes in front of the agent's",
		resolved: prompts.ResolvedAgent{
			Path:   "hound",
			Prompt: prompts.CompiledAgent{Prompt: "You are a hound."},
			Config: prompts.PromptConfig{Model: "openai/gpt-5.5", Prompt: "Answer briefly."},
		},
		want: Agent{
			Model:        "openai/gpt-5.5",
			Instructions: "Answer briefly.\n\nYou are a hound.",
		},
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := AgentFrom(testCase.resolved); got != testCase.want {
				t.Fatalf("AgentFrom() = %#v, want %#v", got, testCase.want)
			}
		})
	}
}
