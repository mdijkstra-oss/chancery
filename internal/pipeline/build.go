package pipeline

import (
	"fmt"

	"github.com/mdijkstra-oss/chancery/internal/messages"
	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/protocol"
)

func BuildRequestParams(agentReference string, req protocol.ChatRequest, registry prompts.Registry) (protocol.RequestParams, prompts.PromptConfig, error) {
	resolved, err := registry.ResolveAgent(agentReference)
	if err != nil {
		return protocol.RequestParams{}, prompts.PromptConfig{}, err
	}
	return BuildRequestParamsForAgent(resolved, req, registry)
}

func BuildRequestParamsForAgent(resolved prompts.ResolvedAgent, req protocol.ChatRequest, registry prompts.Registry) (protocol.RequestParams, prompts.PromptConfig, error) {
	promptCfg := resolved.Config

	expanded := messages.ExpandMessages(req.Messages, registry.Modes)
	expanded = messages.DropEmptyContent(expanded)
	expanded = messages.ReorderToolMessages(expanded)
	expanded, cacheBreakpoints := messages.ExtractCacheBreakpoints(expanded)

	toolNames := protocol.ExtractToolNames(req.Tools)
	toolPrompt, _, err := prompts.LoadToolPrompts(registry.Root, toolNames)
	if err != nil {
		return protocol.RequestParams{}, prompts.PromptConfig{}, fmt.Errorf("load tool prompts: %w", err)
	}

	fullPrompt := resolved.Prompt.Prompt
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
		MaxTokens:        promptCfg.MaxTokens,
		AutoCache:        promptCfg.AutoCache,
		CacheTTL:         promptCfg.CacheTTL,
		Tools:            req.Tools,
		Messages:         expanded,
		ResponseFormat:   req.ResponseFormat,
		CacheBreakpoints: cacheBreakpoints,
	}

	return params, promptCfg, nil
}
