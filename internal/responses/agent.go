package responses

import (
	"github.com/mdijkstra-oss/chancery/internal/prompts"
)

// AgentFrom reads what a resolved agent contributes to a body: the model its alias
// names, the prompt its Markdown compiled to, and the sampling fields beside them.
func AgentFrom(resolved prompts.ResolvedAgent) Agent {
	config := resolved.Config
	return Agent{
		Model:            config.Model,
		Instructions:     instructionsFor(resolved),
		ReasoningEffort:  config.ReasoningEffort,
		ReasoningSummary: config.ReasoningSummary,
		Verbosity:        config.Verbosity,
		ServiceTier:      config.ServiceTier,
		MaxOutputTokens:  config.MaxTokens,
	}
}

// A model's own prompt goes in front of the agent's, so an alias can say how it
// behaves while the Markdown file says what the agent is.
func instructionsFor(resolved prompts.ResolvedAgent) string {
	if resolved.Config.Prompt == "" {
		return resolved.Prompt.Prompt
	}
	return resolved.Config.Prompt + "\n\n" + resolved.Prompt.Prompt
}
