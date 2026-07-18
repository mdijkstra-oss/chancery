package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseManifest(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Line
	}{
		{name: "empty", content: "", want: []Line{{Literal: ""}}},
		{name: "literal", content: "first\nsecond", want: []Line{{Literal: "first"}, {Literal: "second"}}},
		{name: "include", content: "[piece.md]", want: []Line{{Include: "piece.md"}}},
		{name: "nested include", content: "[group/piece.md]", want: []Line{{Include: "group/piece.md"}}},
		{name: "non include", content: "[value]", want: []Line{{Literal: "[value]"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if diff := cmp.Diff(test.want, ParseManifest(test.content)); diff != "" {
				t.Errorf("manifest mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveManifest(t *testing.T) {
	reader := testReader(map[string]string{
		"/shared/group/piece.md": "shared piece\n",
		"/local/piece.md":        "local piece",
	})
	tests := []struct {
		name      string
		lines     []Line
		local     string
		want      string
		wantError bool
	}{
		{name: "shared include", lines: []Line{{Include: "group/piece.md"}}, want: "shared piece"},
		{name: "local include", lines: []Line{{Include: "piece.md"}}, local: "/local", want: "local piece"},
		{name: "literal and include", lines: []Line{{Literal: "start"}, {Include: "group/piece.md"}}, want: "start\nshared piece"},
		{name: "missing include", lines: []Line{{Include: "missing.md"}}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveManifest(test.lines, reader, "/shared", test.local, nil)
			if test.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve manifest: %v", err)
			}
			if got.Prompt != test.want {
				t.Errorf("prompt = %q, want %q", got.Prompt, test.want)
			}
		})
	}
}

func TestManifestKeyFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "top-level file", path: "/config/agent.md", want: "agent"},
		{name: "folder index", path: "/config/folder/index.md", want: "folder"},
		{name: "folder file", path: "/config/folder/agent.md", want: "folder/agent"},
		{name: "nested file", path: "/config/one/two/agent.md", want: "one/two/agent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ManifestKeyFromPath(test.path, "/config"); got != test.want {
				t.Errorf("key = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	root := writeValidConfig(t)
	registry, report := Load(root)
	if report.HasErrors() {
		t.Fatalf("load errors: %v", report.Diagnostics)
	}
	if report.WarningCount() != 1 {
		t.Fatalf("warnings = %d, want 1: %v", report.WarningCount(), report.Diagnostics)
	}
	if diff := cmp.Diff([]string{"embeddings", "folder", "simple"}, registry.AgentPaths()); diff != "" {
		t.Errorf("agent paths mismatch (-want +got):\n%s", diff)
	}
	if registry.ProviderCount() != 2 {
		t.Errorf("provider count = %d, want 2", registry.ProviderCount())
	}
	if registry.ModelCount() != 4 {
		t.Errorf("model count = %d, want 4", registry.ModelCount())
	}
	if got := registry.Agents["folder"].Prompt; got != "local fragment\n\nfolder prompt" {
		t.Errorf("folder prompt = %q", got)
	}
	if got := registry.Modes["planning"]; got != "mode fragment" {
		t.Errorf("planning mode = %q", got)
	}
	if got := registry.Configs["simple"].Provider.APIKey; got != "" {
		t.Errorf("static load resolved API key %q", got)
	}
	if got := registry.Configs["folder"].Model; got != "upstream-fast" {
		t.Errorf("default model = %q, want upstream-fast", got)
	}
	if got := registry.NamedConfigs["folder"]["deep"].ReasoningEffort; got != "high" {
		t.Errorf("named reasoning = %q, want high", got)
	}
}

func TestLoadValidation(t *testing.T) {
	tests := []struct {
		name        string
		agentPath   string
		agent       string
		providers   string
		wantMessage string
		wantWarning bool
	}{
		{
			name:        "malformed providers yaml",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", "prompt"),
			providers:   "providers: [",
			wantMessage: "malformed YAML",
		},
		{
			name:        "malformed frontmatter yaml",
			agentPath:   "agent.md",
			agent:       "---\nmodel: [\n---\nprompt",
			wantMessage: "malformed YAML frontmatter",
		},
		{
			name:        "unknown model",
			agentPath:   "agent.md",
			agent:       agentFile("missing-model", "prompt"),
			wantMessage: "unknown model",
		},
		{
			name:        "missing model definition",
			agentPath:   "agent.md",
			agent:       "---\ndescription: test\n---\nprompt",
			wantMessage: "exactly one of model or models",
		},
		{
			name:        "multi-entry without default",
			agentPath:   "agent.md",
			agent:       "---\nmodels:\n  first:\n    model: model-fast\n  second:\n    model: model-deep\n---\nprompt",
			wantMessage: "require default",
		},
		{
			name:        "default names missing entry",
			agentPath:   "agent.md",
			agent:       "---\nmodels:\n  first:\n    model: model-fast\n  second:\n    model: model-deep\ndefault: missing\n---\nprompt",
			wantMessage: "does not name",
		},
		{
			name:        "missing include",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", "[missing.md]"),
			wantMessage: "include \"missing.md\"",
		},
		{
			name:        "include escapes config",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", "[../../outside.md]"),
			wantMessage: "escapes config directory",
		},
		{
			name:        "model prompt escapes config",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", "body"),
			providers:   "providers:\n  provider-a:\n    protocol: responses\n    base_url: https://provider.example/v1\n    api_key_env: PROVIDER_KEY\n    models:\n      model-fast:\n        prompt: ../../outside.md\n",
			wantMessage: "escapes config directory",
		},
		{
			name:        "agent prompt frontmatter rejected",
			agentPath:   "agent.md",
			agent:       "---\nmodel: model-fast\nprompt: legacy.md\n---\nbody",
			wantMessage: "prompt frontmatter is not supported",
		},
		{
			name:        "named model name contains dot",
			agentPath:   "agent.md",
			agent:       "---\nmodels:\n  invalid.name:\n    model: model-fast\n---\nprompt",
			wantMessage: "must not contain dots or slashes",
		},
		{
			name:        "orphaned markdown",
			agentPath:   "orphan.md",
			agent:       "fragment",
			wantMessage: "orphaned Markdown",
		},
		{
			name:        "empty prompt warning",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", ""),
			wantMessage: "empty prompt body",
			wantWarning: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			providers := test.providers
			if providers == "" {
				providers = validProvidersYAML()
			}
			writeTestFile(t, filepath.Join(root, "providers.yaml"), providers)
			writeTestFile(t, filepath.Join(root, test.agentPath), test.agent)
			_, report := Load(root)
			severity := SeverityError
			if test.wantWarning {
				severity = SeverityWarning
			}
			if !hasDiagnostic(report, severity, test.wantMessage) {
				t.Fatalf("missing %s containing %q: %v", severity, test.wantMessage, report.Diagnostics)
			}
		})
	}
}

func TestResolveAgent(t *testing.T) {
	registry, report := Load(writeValidConfig(t))
	if report.HasErrors() {
		t.Fatalf("load errors: %v", report.Diagnostics)
	}
	tests := []struct {
		name      string
		reference string
		wantPath  string
		wantName  string
		wantModel string
		wantError bool
	}{
		{name: "top-level agent", reference: "simple", wantPath: "simple", wantModel: "upstream-fast"},
		{name: "folder index default", reference: "folder", wantPath: "folder", wantName: "fast", wantModel: "upstream-fast"},
		{name: "explicit named model", reference: "folder.deep", wantPath: "folder", wantName: "deep", wantModel: "upstream-deep"},
		{name: "unknown named model", reference: "folder.missing", wantError: true},
		{name: "unknown agent", reference: "missing", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := registry.ResolveAgent(test.reference)
			if test.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve agent: %v", err)
			}
			if got.Path != test.wantPath || got.Name != test.wantName || got.Config.Model != test.wantModel {
				t.Errorf("resolved = %#v", got)
			}
		})
	}
}

func TestNamedRouteCollision(t *testing.T) {
	root := writeValidConfig(t)
	writeTestFile(t, filepath.Join(root, "folder.fast.md"), agentFile("model-fast", "collision"))
	_, report := Load(root)
	if !hasDiagnostic(report, SeverityError, "collides with named model route") {
		t.Fatalf("missing collision diagnostic: %v", report.Diagnostics)
	}
}

func TestToolPromptCannotEscapeConfig(t *testing.T) {
	root := writeValidConfig(t)
	outside := filepath.Join(t.TempDir(), "outside.md")
	writeTestFile(t, outside, "external prompt")
	toolPath := filepath.Join(root, "tools", "external.search.md")
	if err := os.MkdirAll(filepath.Dir(toolPath), 0755); err != nil {
		t.Fatalf("create tools directory: %v", err)
	}
	if err := os.Symlink(outside, toolPath); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	_, report := Load(root)
	if !hasDiagnostic(report, SeverityError, "escapes config directory through symlink") {
		t.Fatalf("missing symlink diagnostic: %v", report.Diagnostics)
	}
	if _, _, err := LoadToolPrompts(root, []string{"search"}); err == nil {
		t.Fatal("expected tool prompt containment error")
	}
}

func TestToolsDirectoryCannotEscapeConfig(t *testing.T) {
	root := writeValidConfig(t)
	outside := t.TempDir()
	writeTestFile(t, filepath.Join(outside, "external.search.md"), "external prompt")
	if err := os.Symlink(outside, filepath.Join(root, "tools")); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	_, report := Load(root)
	if !hasDiagnostic(report, SeverityError, "escapes config directory through symlink") {
		t.Fatalf("missing tools directory diagnostic: %v", report.Diagnostics)
	}
}

func TestWithAPIKeysRequiresEveryProvider(t *testing.T) {
	registry := Registry{
		Agents:       map[string]CompiledAgent{},
		Configs:      map[string]PromptConfig{},
		NamedConfigs: map[string]map[string]PromptConfig{},
		Defaults:     map[string]string{},
		Descriptions: map[string]string{},
		Modes:        map[string]string{},
		providers: map[string]ProviderConfig{
			"unused": {Key: "unused", APIKeyEnv: "UNUSED_PROVIDER_KEY"},
		},
	}
	if _, err := registry.WithAPIKeys(fixedLookup("")); err == nil {
		t.Fatal("expected missing provider key error")
	}
}

func TestResolveAPIKey(t *testing.T) {
	cfg := PromptConfig{Provider: ProviderConfig{Key: "provider-a", APIKeyEnv: "TEST_PROVIDER_KEY"}}
	tests := []struct {
		name      string
		lookup    func(string) string
		wantKey   string
		wantError bool
	}{
		{name: "key resolved", lookup: fixedLookup("secret"), wantKey: "secret"},
		{name: "missing key", lookup: fixedLookup(""), wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveAPIKey(cfg, test.lookup)
			if test.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve API key: %v", err)
			}
			if got.Provider.APIKey != test.wantKey {
				t.Errorf("API key = %q, want %q", got.Provider.APIKey, test.wantKey)
			}
			if cfg.Provider.APIKey != "" {
				t.Error("input config was mutated")
			}
		})
	}
}

func TestResolveModel(t *testing.T) {
	models := map[string]modelEntry{
		"base":  {Provider: "provider-a", Name: "upstream"},
		"child": {Extends: "base", ServiceTier: "priority"},
	}
	got, err := resolveModel("child", models)
	if err != nil {
		t.Fatalf("resolve model: %v", err)
	}
	want := modelEntry{Provider: "provider-a", Name: "upstream", ServiceTier: "priority"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("model mismatch (-want +got):\n%s", diff)
	}
}

func testReader(files map[string]string) func(string) (string, error) {
	return func(path string) (string, error) {
		content, ok := files[path]
		if !ok {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return content, nil
	}
}

func writeValidConfig(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"providers.yaml":          validProvidersYAML(),
		"simple.md":               agentFile("model-fast", "simple prompt"),
		"embeddings.md":           agentFile("model-vector", ""),
		"folder/index.md":         "---\ndescription: named agent\ntemperature: 0.2\nmodels:\n  fast:\n    model: model-fast\n  deep:\n    model: model-deep\n    reasoning_effort: high\ndefault: fast\n---\n[fragment.md]\n\nfolder prompt",
		"folder/fragment.md":      "local fragment",
		"modes/planning.md":       "[planning/piece.md]",
		"modes/planning/piece.md": "mode fragment",
	}
	for path, content := range files {
		writeTestFile(t, filepath.Join(root, path), content)
	}
	return root
}

func validProvidersYAML() string {
	return `providers:
  provider-a:
    protocol: responses
    base_url: https://provider-a.example/v1
    api_key_env: TEST_PROVIDER_A_KEY
    models:
      model-fast:
        name: upstream-fast
        reasoning_effort: low
      model-vector:
        name: upstream-vector
        type: embedding
        dimensions: 64
  provider-b:
    protocol: anthropic
    base_url: https://provider-b.example
    api_key_env: TEST_PROVIDER_B_KEY
    models:
      model-deep:
        name: upstream-deep
        reasoning_effort: medium
`
}

func agentFile(model, body string) string {
	return fmt.Sprintf("---\ndescription: test agent\nmodel: %s\n---\n%s", model, body)
}

func hasDiagnostic(report Report, severity Severity, message string) bool {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Severity == severity && strings.Contains(diagnostic.Message, message) {
			return true
		}
	}
	return false
}

func fixedLookup(value string) func(string) string {
	return func(string) string {
		return value
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
