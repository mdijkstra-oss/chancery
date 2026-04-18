package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func testReader(files map[string]string) func(string) (string, error) {
	return func(path string) (string, error) {
		content, ok := files[path]
		if !ok {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return content, nil
	}
}

func TestParseManifest(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []Line
	}{
		{
			name:    "empty",
			content: "",
			want:    []Line{{Literal: ""}},
		},
		{
			name:    "literal only",
			content: "Hello world\nSecond line",
			want:    []Line{{Literal: "Hello world"}, {Literal: "Second line"}},
		},
		{
			name:    "include only",
			content: "[nabu/identity.md]",
			want:    []Line{{Include: "nabu/identity.md"}},
		},
		{
			name:    "mixed",
			content: "[nabu/identity.md]\n\nYou are helpful.\n\n[planning/planning.md]",
			want: []Line{
				{Include: "nabu/identity.md"},
				{Literal: ""},
				{Literal: "You are helpful."},
				{Literal: ""},
				{Include: "planning/planning.md"},
			},
		},
		{
			name:    "whitespace around include",
			content: "  [nabu/identity.md]  ",
			want:    []Line{{Include: "nabu/identity.md"}},
		},
		{
			name:    "bracket in literal not matching pattern",
			content: "[not-an-include]",
			want:    []Line{{Literal: "[not-an-include]"}},
		},
		{
			name:    "nested path include",
			content: "[qualitative-researcher/coding.md]",
			want:    []Line{{Include: "qualitative-researcher/coding.md"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseManifest(tc.content)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveManifest(t *testing.T) {
	shared := "/shared"
	files := map[string]string{
		"/shared/nabu/identity.md":      "I am Nabu.",
		"/shared/nabu/discipline.md":    "Be disciplined.",
		"/shared/planning/planning.md":  "Plan carefully.",
		"/shared/chat/style.md":         "Be direct.",
	}
	reader := testReader(files)

	cases := []struct {
		name         string
		lines        []Line
		wantPrompt   string
		wantSources  []string
		wantSegments []Segment
		wantErr      bool
	}{
		{
			name:        "all includes resolved",
			lines:       []Line{{Include: "nabu/identity.md"}, {Include: "nabu/discipline.md"}},
			wantPrompt:  "I am Nabu.\nBe disciplined.",
			wantSources: []string{"nabu/identity.md", "nabu/discipline.md"},
			wantSegments: []Segment{
				{Source: "nabu/identity.md", Content: "I am Nabu."},
				{Source: "nabu/discipline.md", Content: "Be disciplined."},
			},
		},
		{
			name:    "missing include errors",
			lines:   []Line{{Include: "nonexistent.md"}},
			wantErr: true,
		},
		{
			name: "mixed content with glue text",
			lines: []Line{
				{Include: "nabu/identity.md"},
				{Literal: ""},
				{Literal: "You understand this methodology."},
				{Literal: ""},
				{Include: "planning/planning.md"},
			},
			wantPrompt:  "I am Nabu.\n\nYou understand this methodology.\n\nPlan carefully.",
			wantSources: []string{"nabu/identity.md", "planning/planning.md"},
			wantSegments: []Segment{
				{Source: "nabu/identity.md", Content: "I am Nabu."},
				{Content: ""},
				{Content: "You understand this methodology."},
				{Content: ""},
				{Source: "planning/planning.md", Content: "Plan carefully."},
			},
		},
		{
			name:        "literal only",
			lines:       []Line{{Literal: "Just text."}, {Literal: "More text."}},
			wantPrompt:  "Just text.\nMore text.",
			wantSources: nil,
			wantSegments: []Segment{
				{Content: "Just text."},
				{Content: "More text."},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveManifest(tc.lines, reader, shared, "", nil)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantPrompt, got.Prompt); diff != "" {
				t.Errorf("prompt mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSources, got.Sources); diff != "" {
				t.Errorf("sources mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSegments, got.Segments); diff != "" {
				t.Errorf("segments mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("skip keeps placeholder", func(t *testing.T) {
		lines := []Line{
			{Include: "nabu/identity.md"},
			{Literal: ""},
			{Include: "chat/style.md"},
		}
		skipChat := func(path string) bool { return strings.Contains(path, "chat/") }
		got, err := ResolveManifest(lines, reader, shared, "", skipChat)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantPrompt := "I am Nabu.\n\n[chat/style.md]"
		if diff := cmp.Diff(wantPrompt, got.Prompt); diff != "" {
			t.Errorf("prompt mismatch (-want +got):\n%s", diff)
		}
		wantSources := []string{"nabu/identity.md"}
		if diff := cmp.Diff(wantSources, got.Sources); diff != "" {
			t.Errorf("sources mismatch (-want +got):\n%s", diff)
		}
		wantSegments := []Segment{
			{Source: "nabu/identity.md", Content: "I am Nabu."},
			{Content: ""},
			{Content: "[chat/style.md]"},
		}
		if diff := cmp.Diff(wantSegments, got.Segments); diff != "" {
			t.Errorf("segments mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("local dir takes precedence over shared", func(t *testing.T) {
		merged := testReader(map[string]string{
			"/shared/nabu/identity.md": "I am Nabu.",
			"/shared/guide.md":         "Shared guide.",
			"/local/guide.md":          "Local guide.",
		})
		lines := []Line{
			{Include: "nabu/identity.md"},
			{Literal: ""},
			{Include: "guide.md"},
		}
		got, err := ResolveManifest(lines, merged, shared, "/local", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantPrompt := "I am Nabu.\n\nLocal guide."
		if diff := cmp.Diff(wantPrompt, got.Prompt); diff != "" {
			t.Errorf("prompt mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("falls back to shared when not in local", func(t *testing.T) {
		lines := []Line{{Include: "nabu/identity.md"}}
		got, err := ResolveManifest(lines, reader, shared, "/nonexistent", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantPrompt := "I am Nabu."
		if diff := cmp.Diff(wantPrompt, got.Prompt); diff != "" {
			t.Errorf("prompt mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestManifestKeyFromPath(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		agentsDir string
		want      string
	}{
		{
			name:      "index.md becomes parent dir",
			path:      "/agents/qual-coder/index.md",
			agentsDir: "/agents",
			want:      "qual-coder",
		},
		{
			name:      "named file becomes parent/name",
			path:      "/agents/qual-coder/plan.md",
			agentsDir: "/agents",
			want:      "qual-coder/plan",
		},
		{
			name:      "top-level index",
			path:      "/agents/compacter/index.md",
			agentsDir: "/agents",
			want:      "compacter",
		},
		{
			name:      "nested path named file",
			path:      "/agents/nested/sub-agent/compact.md",
			agentsDir: "/agents",
			want:      "nested/sub-agent/compact",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ManifestKeyFromPath(tc.path, tc.agentsDir)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCompileRegistry(t *testing.T) {
	base := t.TempDir()
	promptsDir := filepath.Join(base, "prompts")

	sharedFiles := map[string]string{
		"shared/nabu/identity.md":   "I am Nabu.",
		"shared/nabu/discipline.md": "Be disciplined.",
		"shared/chat/style.md":      "Be direct.",
	}
	modeIncludeFiles := map[string]string{
		"modes/planning/planning.md":   "Plan carefully.",
		"modes/planning/template.md":   "Template here.",
		"modes/execution/execution.md": "Execute the plan.",
	}
	for name, content := range sharedFiles {
		writeTestFile(t, filepath.Join(promptsDir, name), content)
	}
	for name, content := range modeIncludeFiles {
		writeTestFile(t, filepath.Join(promptsDir, name), content)
	}

	agentFiles := map[string]string{
		"agents/qual-coder/index.md":    "[nabu/identity.md]\n[nabu/discipline.md]\n\nYou are an expert.",
		"agents/qual-coder/plan.md":     "[nabu/identity.md]\n\nPlan the work.\n\n[chat/style.md]",
		"agents/compacter/index.md":     "[nabu/identity.md]\n\nYou compact.",
		"agents/advisor/index.md":       "[requirements.md]\n\nYou advise.",
		"agents/advisor/requirements.md": "Plan well.",
	}
	for name, content := range agentFiles {
		writeTestFile(t, filepath.Join(promptsDir, name), content)
	}

	t.Setenv("TEST_API_KEY", "test-key-123")

	providerJSON := `{
		"protocol": "responses",
		"base_url": "https://api.openai.com/v1",
		"api_key_env": "TEST_API_KEY",
		"models": {
			"gpt-5.2": {"name": "gpt-5.2", "pricing": {"input": 0, "output": 0, "cached_input": 0}},
			"gpt-5-mini": {"name": "gpt-5-mini", "pricing": {"input": 0, "output": 0, "cached_input": 0}}
		}
	}`
	writeTestFile(t, filepath.Join(promptsDir, "config", "openai.json"), providerJSON)

	agentsJSON := `{
		"qual-coder": {"model": "gpt-5.2", "reasoning_effort": "high", "verbosity": "low"},
		"compacter": {"model": "gpt-5.2", "reasoning_effort": "medium"},
		"advisor": {"model": "gpt-5-mini", "reasoning_effort": "low"}
	}`
	writeTestFile(t, filepath.Join(promptsDir, "config", "agents.json"), agentsJSON)

	modeFiles := map[string]string{
		"modes/planning.md":  "Planning intro.\n\n[planning/planning.md]\n[planning/template.md]",
		"modes/execution.md": "Execution intro.\n\n[execution/execution.md]",
	}
	for name, content := range modeFiles {
		writeTestFile(t, filepath.Join(promptsDir, name), content)
	}

	registry := CompileRegistry(promptsDir)

	agentCases := []struct {
		key        string
		wantPrompt string
	}{
		{
			key:        "qual-coder",
			wantPrompt: "I am Nabu.\nBe disciplined.\n\nYou are an expert.",
		},
		{
			key:        "qual-coder/plan",
			wantPrompt: "I am Nabu.\n\nPlan the work.\n\nBe direct.",
		},
		{
			key:        "compacter",
			wantPrompt: "I am Nabu.\n\nYou compact.",
		},
		{
			key:        "advisor",
			wantPrompt: "Plan well.\n\nYou advise.",
		},
	}

	for _, tc := range agentCases {
		t.Run("agent/"+tc.key, func(t *testing.T) {
			agent, ok := registry.Agents[tc.key]
			if !ok {
				t.Fatalf("agent %q not found in registry; keys: %v", tc.key, registryKeys(registry.Agents))
			}
			if diff := cmp.Diff(tc.wantPrompt, agent.Prompt); diff != "" {
				t.Errorf("prompt mismatch (-want +got):\n%s", diff)
			}
		})
	}

	testProvider := ProviderConfig{Key: "openai", Protocol: ProtocolResponses, BaseURL: "https://api.openai.com/v1", APIKey: "test-key-123"}

	configCases := []struct {
		key  string
		want PromptConfig
	}{
		{
			key:  "qual-coder",
			want: PromptConfig{Model: "gpt-5.2", ReasoningEffort: "high", Verbosity: "low", Provider: testProvider},
		},
		{
			key:  "qual-coder/plan",
			want: PromptConfig{Model: "gpt-5.2", ReasoningEffort: "high", Verbosity: "low", Provider: testProvider},
		},
		{
			key:  "compacter",
			want: PromptConfig{Model: "gpt-5.2", ReasoningEffort: "medium", Provider: testProvider},
		},
	}

	for _, tc := range configCases {
		t.Run("config/"+tc.key, func(t *testing.T) {
			cfg, ok := registry.Configs[tc.key]
			if !ok {
				t.Fatalf("config %q not found", tc.key)
			}
			if diff := cmp.Diff(tc.want, cfg); diff != "" {
				t.Errorf("config mismatch (-want +got):\n%s", diff)
			}
		})
	}

	modeCases := []struct {
		key        string
		wantPrompt string
	}{
		{
			key:        "planning",
			wantPrompt: "Planning intro.\n\nPlan carefully.\nTemplate here.",
		},
		{
			key:        "execution",
			wantPrompt: "Execution intro.\n\nExecute the plan.",
		},
	}

	for _, tc := range modeCases {
		t.Run("mode/"+tc.key, func(t *testing.T) {
			prompt, ok := registry.Modes[tc.key]
			if !ok {
				t.Fatalf("mode %q not found in registry", tc.key)
			}
			if diff := cmp.Diff(tc.wantPrompt, prompt); diff != "" {
				t.Errorf("mode prompt mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseToolRequirement(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "plain markdown",
			filename: "01-identity.md",
			want:     "",
		},
		{
			name:     "tool-gated file",
			filename: "02-coding.patch_json_block.md",
			want:     "patch_json_block",
		},
		{
			name:     "tool prompt renamed",
			filename: "orientation.orientate.md",
			want:     "orientate",
		},
		{
			name:     "hyphenated base with tool",
			filename: "blocks-jq.run_local_shell.md",
			want:     "run_local_shell",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseToolRequirement(tc.filename)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMergeConfig(t *testing.T) {
	prov := ProviderConfig{Protocol: ProtocolResponses, BaseURL: "https://api.openai.com/v1", APIKey: "k"}

	cases := []struct {
		name     string
		model    modelEntry
		agent    agentEntry
		provider ProviderConfig
		want     PromptConfig
	}{
		{
			name:     "model defaults only",
			model:    modelEntry{Name: "gpt-5-mini", Pricing: Pricing{Input: 0.25, Output: 2.00, CachedInput: 0.025}},
			agent:    agentEntry{Model: "gpt-5-mini"},
			provider: prov,
			want:     PromptConfig{Model: "gpt-5-mini", Pricing: Pricing{Input: 0.25, Output: 2.00, CachedInput: 0.025}, Provider: prov},
		},
		{
			name:     "agent overrides reasoning",
			model:    modelEntry{Name: "gpt-5-mini", ReasoningEffort: "off", Pricing: Pricing{Input: 0.25}},
			agent:    agentEntry{Model: "gpt-5-mini", ReasoningEffort: "low"},
			provider: prov,
			want:     PromptConfig{Model: "gpt-5-mini", ReasoningEffort: "low", Pricing: Pricing{Input: 0.25}, Provider: prov},
		},
		{
			name:     "priority variant carries own pricing and tier",
			model:    modelEntry{Name: "gpt-5-mini", ServiceTier: "priority", Pricing: Pricing{Input: 0.45, Output: 3.60, CachedInput: 0.045}},
			agent:    agentEntry{Model: "gpt-5-mini-prio"},
			provider: prov,
			want:     PromptConfig{Model: "gpt-5-mini", ServiceTier: "priority", Pricing: Pricing{Input: 0.45, Output: 3.60, CachedInput: 0.045}, Provider: prov},
		},
		{
			name:     "embedding model with dimensions",
			model:    modelEntry{Name: "text-embedding-3-large", Type: "embedding", Dimensions: 1024},
			agent:    agentEntry{Model: "text-embedding-3-large"},
			provider: prov,
			want:     PromptConfig{Model: "text-embedding-3-large", Dimensions: 1024, Provider: prov},
		},
		{
			name:     "agent overrides compact_at",
			model:    modelEntry{Name: "gpt-5.4", CompactAt: 200000, Pricing: Pricing{Input: 2.50}},
			agent:    agentEntry{Model: "gpt-5.4", CompactAt: 300000},
			provider: prov,
			want:     PromptConfig{Model: "gpt-5.4", CompactAt: 300000, Pricing: Pricing{Input: 2.50}, Provider: prov},
		},
		{
			name:     "temperature from agent",
			model:    modelEntry{Name: "gpt-4o-mini", Pricing: Pricing{Input: 0.15}},
			agent:    agentEntry{Model: "gpt-4o-mini", Temperature: float64Ptr(0)},
			provider: prov,
			want:     PromptConfig{Model: "gpt-4o-mini", Temperature: float64Ptr(0), Pricing: Pricing{Input: 0.15}, Provider: prov},
		},
		{
			name: "all model fields inherited",
			model: modelEntry{
				Name:             "gpt-5.4",
				ReasoningEffort:  "none",
				ReasoningSummary: "auto",
				Verbosity:        "low",
				ServiceTier:      "default",
				CompactAt:        300000,
				Pricing:          Pricing{Input: 2.50, Output: 15.00, CachedInput: 0.25},
			},
			agent:    agentEntry{Model: "gpt-5.4"},
			provider: prov,
			want: PromptConfig{
				Model:            "gpt-5.4",
				ReasoningEffort:  "none",
				ReasoningSummary: "auto",
				Verbosity:        "low",
				ServiceTier:      "default",
				CompactAt:        300000,
				Pricing:          Pricing{Input: 2.50, Output: 15.00, CachedInput: 0.25},
				Provider:         prov,
			},
		},
		{
			name:     "agent overrides model service_tier",
			model:    modelEntry{Name: "gpt-5.4", Pricing: Pricing{Input: 2.50}},
			agent:    agentEntry{Model: "gpt-5.4", ServiceTier: "default"},
			provider: prov,
			want:     PromptConfig{Model: "gpt-5.4", ServiceTier: "default", Pricing: Pricing{Input: 2.50}, Provider: prov},
		},
		{
			name:     "seed from agent",
			model:    modelEntry{Name: "gemini-2.5-flash-lite"},
			agent:    agentEntry{Model: "gemini-2.5-flash-lite", Seed: true},
			provider: ProviderConfig{Protocol: ProtocolGemini, BaseURL: "https://gen.googleapis.com/v1beta", APIKey: "k"},
			want:     PromptConfig{Model: "gemini-2.5-flash-lite", Seed: true, Provider: ProviderConfig{Protocol: ProtocolGemini, BaseURL: "https://gen.googleapis.com/v1beta", APIKey: "k"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeConfig(tc.model, tc.agent, tc.provider)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveModel(t *testing.T) {
	base := modelEntry{Provider: "openai", Name: "gpt-5-mini", Pricing: Pricing{Input: 0.25, Output: 2.00, CachedInput: 0.025}}
	prio := modelEntry{Extends: "gpt-5-mini", ServiceTier: "priority", Pricing: Pricing{Input: 0.45, Output: 3.60, CachedInput: 0.045}}

	cases := []struct {
		name   string
		key    string
		models map[string]modelEntry
		want   modelEntry
	}{
		{
			name:   "no extends returns as-is",
			key:    "gpt-5-mini",
			models: map[string]modelEntry{"gpt-5-mini": base},
			want:   base,
		},
		{
			name:   "single extends inherits name and overrides fields",
			key:    "gpt-5-mini-prio",
			models: map[string]modelEntry{"gpt-5-mini": base, "gpt-5-mini-prio": prio},
			want:   modelEntry{Provider: "openai", Name: "gpt-5-mini", ServiceTier: "priority", Pricing: Pricing{Input: 0.45, Output: 3.60, CachedInput: 0.045}},
		},
		{
			name: "extends inherits pricing when child has none",
			key:  "gpt-5-mini-prio",
			models: map[string]modelEntry{
				"gpt-5-mini":      base,
				"gpt-5-mini-prio": {Extends: "gpt-5-mini", ServiceTier: "priority"},
			},
			want: modelEntry{Provider: "openai", Name: "gpt-5-mini", ServiceTier: "priority", Pricing: Pricing{Input: 0.25, Output: 2.00, CachedInput: 0.025}},
		},
		{
			name: "two-level chain",
			key:  "c",
			models: map[string]modelEntry{
				"a": {Provider: "openai", Name: "base-model", Pricing: Pricing{Input: 1.00}},
				"b": {Extends: "a", ServiceTier: "priority"},
				"c": {Extends: "b", Pricing: Pricing{Input: 9.99, Output: 9.99, CachedInput: 9.99}},
			},
			want: modelEntry{Provider: "openai", Name: "base-model", ServiceTier: "priority", Pricing: Pricing{Input: 9.99, Output: 9.99, CachedInput: 9.99}},
		},
		{
			name: "child overrides provider",
			key:  "b",
			models: map[string]modelEntry{
				"a": {Provider: "openai", Name: "base-model", Pricing: Pricing{Input: 1.00}},
				"b": {Extends: "a", Provider: "gemini"},
			},
			want: modelEntry{Provider: "gemini", Name: "base-model", Pricing: Pricing{Input: 1.00}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveModel(tc.key, tc.models)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveModelPanics(t *testing.T) {
	cases := []struct {
		name   string
		key    string
		models map[string]modelEntry
	}{
		{
			name:   "unknown extends target",
			key:    "child",
			models: map[string]modelEntry{"child": {Extends: "nonexistent"}},
		},
		{
			name: "cycle detection",
			key:  "a",
			models: map[string]modelEntry{
				"a": {Extends: "b"},
				"b": {Extends: "a"},
			},
		},
		{
			name: "depth exceeded",
			key:  "g",
			models: map[string]modelEntry{
				"a": {Name: "x"},
				"b": {Extends: "a"},
				"c": {Extends: "b"},
				"d": {Extends: "c"},
				"e": {Extends: "d"},
				"f": {Extends: "e"},
				"g": {Extends: "f"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic, got none")
				}
			}()
			resolveModel(tc.key, tc.models)
		})
	}
}

func TestResolveProviders(t *testing.T) {
	t.Setenv("TEST_OPENAI_KEY", "sk-openai")
	t.Setenv("TEST_GEMINI_KEY", "sk-gemini")

	entries := map[string]ProviderEntry{
		"openai": {Protocol: ProtocolResponses, BaseURL: "https://api.openai.com/v1", APIKeyEnv: "TEST_OPENAI_KEY"},
		"gemini": {Protocol: ProtocolGemini, BaseURL: "https://generativelanguage.googleapis.com/v1beta", APIKeyEnv: "TEST_GEMINI_KEY"},
	}

	got := resolveProviders(entries)

	cases := []struct {
		key  string
		want ProviderConfig
	}{
		{
			key:  "openai",
			want: ProviderConfig{Key: "openai", Protocol: ProtocolResponses, BaseURL: "https://api.openai.com/v1", APIKey: "sk-openai"},
		},
		{
			key:  "gemini",
			want: ProviderConfig{Key: "gemini", Protocol: ProtocolGemini, BaseURL: "https://generativelanguage.googleapis.com/v1beta", APIKey: "sk-gemini"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, got[tc.key]); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveProvidersPanics(t *testing.T) {
	cases := []struct {
		name    string
		entries map[string]ProviderEntry
		envKey  string
		envVal  string
	}{
		{
			name:    "unknown protocol",
			entries: map[string]ProviderEntry{"bad": {Protocol: "grpc", BaseURL: "http://x", APIKeyEnv: "K"}},
			envKey:  "K",
			envVal:  "v",
		},
		{
			name:    "empty api_key_env",
			entries: map[string]ProviderEntry{"bad": {Protocol: ProtocolResponses, BaseURL: "http://x", APIKeyEnv: ""}},
		},
		{
			name:    "env var not set",
			entries: map[string]ProviderEntry{"bad": {Protocol: ProtocolResponses, BaseURL: "http://x", APIKeyEnv: "UNSET_VAR_12345"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envVal)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic, got none")
				}
			}()
			resolveProviders(tc.entries)
		})
	}
}

func TestValidateSeedProtocols(t *testing.T) {
	gemini := ProviderConfig{Protocol: ProtocolGemini, BaseURL: "https://gen.googleapis.com/v1beta", APIKey: "k"}
	openai := ProviderConfig{Key: "openai", Protocol: ProtocolResponses, BaseURL: "https://api.openai.com/v1", APIKey: "k"}

	t.Run("gemini with seed passes", func(t *testing.T) {
		configs := map[string]PromptConfig{
			"a": {Model: "gemini-2.5-flash-lite", Seed: true, Provider: gemini},
		}
		validateSeedProtocols(configs)
	})

	t.Run("openai without seed passes", func(t *testing.T) {
		configs := map[string]PromptConfig{
			"a": {Model: "gpt-5-mini", Provider: openai},
		}
		validateSeedProtocols(configs)
	})

	t.Run("openai with seed panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, got none")
			}
		}()
		configs := map[string]PromptConfig{
			"bad-agent": {Model: "gpt-5-mini", Seed: true, Provider: openai},
		}
		validateSeedProtocols(configs)
	})
}

func float64Ptr(v float64) *float64 {
	return &v
}

func registryKeys(m map[string]CompiledAgent) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}
