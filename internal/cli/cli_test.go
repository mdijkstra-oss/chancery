package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"hermes-logos/internal/prompts"
)

func TestRunRequiresConfig(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "serve", args: []string{"serve"}},
		{name: "validate", args: []string{"validate"}},
		{name: "list", args: []string{"list"}},
		{name: "call", args: []string{"call", "agent"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := Run(context.Background(), test.args, strings.NewReader(""), &bytes.Buffer{}, &stderr)
			if code == 0 {
				t.Fatal("exit code = 0, want failure")
			}
			if !strings.Contains(stderr.String(), "--config PATH is required") {
				t.Errorf("stderr = %q", stderr.String())
			}
		})
	}
}

func TestRunValidateMissingDirectory(t *testing.T) {
	var stdout bytes.Buffer
	code := Run(context.Background(), []string{"--config", "/path/that/does/not/exist", "validate"}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if code == 0 {
		t.Fatal("exit code = 0, want failure")
	}
	if !strings.Contains(stdout.String(), "config invalid") {
		t.Errorf("stdout = %q", stdout.String())
	}
}

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
		args     []string
		contains string
		json     bool
	}{
		{name: "human", contains: ".fast (default)"},
		{name: "json", args: []string{"--json"}, json: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			code := runList(test.args, registry, prompts.Report{}, &stdout, &bytes.Buffer{})
			if code != 0 {
				t.Fatalf("exit code = %d", code)
			}
			if test.json {
				var output listOutput
				if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
					t.Fatalf("decode JSON: %v", err)
				}
				if output.Summary.Agents != 2 || output.Summary.Models != 3 || output.Summary.Providers != 1 {
					t.Errorf("summary = %#v", output.Summary)
				}
				return
			}
			if !strings.Contains(stdout.String(), test.contains) {
				t.Errorf("stdout = %q", stdout.String())
			}
		})
	}
}

func TestReadInput(t *testing.T) {
	got, err := readInput("", strings.NewReader("from stdin"))
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	if got != "from stdin" {
		t.Errorf("input = %q", got)
	}
}
