package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/matthijn/hermes-logos/internal/prompts"
	"github.com/matthijn/hermes-logos/internal/protocol"
)

func TestBuildRequestParams(t *testing.T) {
	root := t.TempDir()
	toolPath := filepath.Join(root, "tools", "guidance.search.md")
	if err := os.MkdirAll(filepath.Dir(toolPath), 0755); err != nil {
		t.Fatalf("create tools directory: %v", err)
	}
	if err := os.WriteFile(toolPath, []byte("tool guidance"), 0644); err != nil {
		t.Fatalf("write tool prompt: %v", err)
	}

	provider := prompts.ProviderConfig{Key: "provider-a", Protocol: prompts.ProtocolResponses}
	registry := prompts.Registry{
		Root:   root,
		Agents: map[string]prompts.CompiledAgent{"folder": {Prompt: "agent prompt"}},
		Configs: map[string]prompts.PromptConfig{
			"folder": {Model: "upstream-fast", Provider: provider},
		},
		NamedConfigs: map[string]map[string]prompts.PromptConfig{
			"folder": {
				"fast": {Model: "upstream-fast", Provider: provider},
				"deep": {Model: "upstream-deep", ReasoningEffort: "high", Provider: provider},
			},
		},
		Defaults: map[string]string{"folder": "fast"},
		Modes:    map[string]string{},
	}
	req := protocol.ChatRequest{
		Messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"input"}`)},
		Tools:    []json.RawMessage{json.RawMessage(`{"name":"search"}`)},
	}

	resolved, err := registry.ResolveAgent("folder.deep")
	if err != nil {
		t.Fatalf("resolve agent: %v", err)
	}
	params, cfg, err := BuildRequestParamsForAgent(resolved, req, registry)
	if err != nil {
		t.Fatalf("build request params: %v", err)
	}
	if params.Model != "upstream-deep" || cfg.Model != "upstream-deep" {
		t.Errorf("models = %q, %q", params.Model, cfg.Model)
	}
	if params.ReasoningEffort != "high" {
		t.Errorf("reasoning effort = %q, want high", params.ReasoningEffort)
	}
	if params.SystemPrompt != "agent prompt\n\ntool guidance" {
		t.Errorf("system prompt = %q", params.SystemPrompt)
	}
}
