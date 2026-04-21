package pipeline

import (
	"fmt"
	"path/filepath"

	"hermes-logos/internal/messages"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
)

func BuildRequestParams(agentName string, req protocol.ChatRequest, registry prompts.Registry) (protocol.RequestParams, prompts.PromptConfig, error) {
	agent, ok := registry.Agents[agentName]
	if !ok {
		return protocol.RequestParams{}, prompts.PromptConfig{}, fmt.Errorf("unknown agent: %s", agentName)
	}

	promptCfg, ok := registry.Configs[agentName]
	if !ok {
		return protocol.RequestParams{}, prompts.PromptConfig{}, fmt.Errorf("no config for agent: %s", agentName)
	}

	expanded := messages.ExpandMessages(req.Messages, registry.Modes)
	expanded = messages.ExpandApproaches(expanded, registry.Approaches.Entries)

	toolNames := protocol.ExtractToolNames(req.Tools)
	toolPrompt, _, err := prompts.LoadToolPrompts(filepath.Join(prompts.PromptsDir, "tools"), toolNames)
	if err != nil {
		return protocol.RequestParams{}, prompts.PromptConfig{}, fmt.Errorf("load tool prompts: %w", err)
	}

	fullPrompt := agent.Prompt
	if promptCfg.Prompt != "" {
		fullPrompt = promptCfg.Prompt + "\n\n" + fullPrompt
	}
	if toolPrompt != "" {
		fullPrompt = fullPrompt + "\n\n" + toolPrompt
	}

	params := protocol.RequestParams{
		Model:            promptCfg.Model,
		SystemPrompt:     fullPrompt,
		ReasoningEffort:  promptCfg.ReasoningEffort,
		ReasoningSummary: promptCfg.ReasoningSummary,
		Verbosity:        promptCfg.Verbosity,
		ServiceTier:      promptCfg.ServiceTier,
		LegacyThinking:   promptCfg.LegacyThinking,
		Temperature:      promptCfg.Temperature,
		Seed:             promptCfg.Seed,
		Tools:            req.Tools,
		Messages:         expanded,
		ResponseFormat:   req.ResponseFormat,
	}

	return params, promptCfg, nil
}
