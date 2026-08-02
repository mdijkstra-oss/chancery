package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mdijkstra-oss/chancery/internal/prompts"
)

func TestRunList(t *testing.T) {
	provider := prompts.ProviderConfig{Key: "provider-a"}
	registry := prompts.Registry{
		Agents: map[string]prompts.CompiledAgent{
			"named":  {},
			"simple": {},
		},
		Configs: map[string]prompts.PromptConfig{
			"named":  {Model: "upstream-fast", Provider: provider},
			"simple": {Model: "upstream-simple", ReasoningEffort: "low", Provider: provider},
		},
		NamedConfigs: map[string]map[string]prompts.PromptConfig{
			"named": {
				"deep": {Model: "upstream-deep", ReasoningEffort: "high", Provider: provider},
				"fast": {Model: "upstream-fast", ReasoningEffort: "low", Provider: provider},
			},
		},
		Defaults:     map[string]string{"named": "fast"},
		Descriptions: map[string]string{},
		ProviderKeys: []string{"provider-a"},
	}
	tests := []struct {
		name     string
		asJSON   bool
		contains string
	}{
		{name: "human", contains: ".fast (default)"},
		{name: "json", asJSON: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := runList(registry, test.asJSON, &output); err != nil {
				t.Fatalf("run list: %v", err)
			}
			if test.asJSON {
				var decoded listOutput
				if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
					t.Fatalf("decode JSON: %v", err)
				}
				if decoded.Summary.Agents != 2 || decoded.Summary.Models != 3 || decoded.Summary.Providers != 1 {
					t.Errorf("summary = %#v", decoded.Summary)
				}
				return
			}
			if !strings.Contains(output.String(), test.contains) {
				t.Errorf("output = %q", output.String())
			}
		})
	}
}
