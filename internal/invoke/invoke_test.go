package invoke

import (
	"strings"
	"testing"

	"github.com/matthijn/hermes-logos/internal/prompts"
)

func TestResolve(t *testing.T) {
	provider := prompts.ProviderConfig{Key: "provider", APIKeyEnv: "PROVIDER_KEY"}
	registry := prompts.Registry{
		Agents: map[string]prompts.CompiledAgent{
			"assistant":  {},
			"embeddings": {},
		},
		Configs: map[string]prompts.PromptConfig{
			"assistant":  {Provider: provider},
			"embeddings": {Provider: provider},
		},
	}
	tests := []struct {
		name      string
		reference string
		lookup    func(string) string
		wantKind  Kind
		wantError string
	}{
		{name: "unknown agent", reference: "missing", lookup: emptyLookup, wantError: "unknown agent: missing"},
		{name: "missing API key", reference: "assistant", lookup: emptyLookup, wantError: "environment variable PROVIDER_KEY is not set"},
		{name: "chat", reference: "assistant", lookup: populatedLookup, wantKind: KindChat},
		{name: "embeddings", reference: "embeddings", lookup: populatedLookup, wantKind: KindEmbeddings},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target, err := Resolve(test.reference, registry, test.lookup)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Errorf("error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if target.Kind != test.wantKind {
				t.Errorf("kind = %q, want %q", target.Kind, test.wantKind)
			}
		})
	}
}

func emptyLookup(string) string {
	return ""
}

func populatedLookup(string) string {
	return "secret"
}
