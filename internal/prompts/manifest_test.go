package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	registry, report := Load(root, DefaultModelsFile)
	if report.HasErrors() {
		t.Fatalf("load errors: %v", report.Diagnostics)
	}
	if report.WarningCount() != 0 {
		t.Fatalf("warnings = %d, want 0: %v", report.WarningCount(), report.Diagnostics)
	}
	if diff := cmp.Diff([]string{"folder", "prio", "simple"}, registry.AgentPaths()); diff != "" {
		t.Errorf("agent paths mismatch (-want +got):\n%s", diff)
	}
	if registry.ModelCount() != 4 {
		t.Errorf("model count = %d, want 4", registry.ModelCount())
	}
	if got := registry.Agents["folder"].Prompt; got != "local fragment\n\nfolder prompt" {
		t.Errorf("folder prompt = %q", got)
	}
	if got := registry.Configs["folder"].Model; got != "openai/upstream-fast" {
		t.Errorf("default model = %q, want openai/upstream-fast", got)
	}
	if got := registry.NamedConfigs["folder"]["deep"].ReasoningEffort; got != "high" {
		t.Errorf("named reasoning = %q, want high", got)
	}
	prio := registry.Configs["prio"]
	if prio.Model != "openai/upstream-fast" || prio.ServiceTier != "priority" {
		t.Errorf("extended alias = %#v", prio)
	}
}

func TestLoadValidation(t *testing.T) {
	tests := []struct {
		name        string
		agentPath   string
		agent       string
		models      string
		wantMessage string
		wantWarning bool
	}{
		{
			name:        "malformed models yaml",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", "prompt"),
			models:      "models: [",
			wantMessage: "malformed YAML",
		},
		{
			name:        "alias defined twice",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", "prompt"),
			models:      "models:\n  model-fast:\n    model: openai/one\n  model-fast:\n    model: openai/two\n",
			wantMessage: `mapping key "model-fast" already defined`,
		},
		{
			name:        "alias without model",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", "prompt"),
			models:      "models:\n  model-fast:\n    reasoning_effort: low\n",
			wantMessage: `alias "model-fast" has no model`,
		},
		{
			name:        "alias extends unknown alias",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", "prompt"),
			models:      "models:\n  model-fast:\n    extends: missing\n",
			wantMessage: `extends unknown alias "missing"`,
		},
		{
			name:        "temperature names no body field an agent may pin",
			agentPath:   "agent.md",
			agent:       agentFile("model-fast", "prompt"),
			models:      "models:\n  model-fast:\n    model: openai/upstream-fast\n    temperature: 0.2\n",
			wantMessage: `unknown field "temperature"`,
		},
		{
			name:        "malformed frontmatter yaml",
			agentPath:   "agent.md",
			agent:       "---\nmodel: [\n---\nprompt",
			wantMessage: "malformed YAML frontmatter",
		},
		{
			name:        "unknown alias",
			agentPath:   "agent.md",
			agent:       agentFile("missing-model", "prompt"),
			wantMessage: `unknown model alias "missing-model"`,
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
			models:      "models:\n  model-fast:\n    model: openai/upstream-fast\n    prompt: ../../outside.md\n",
			wantMessage: "escapes config directory",
		},
		{
			name:        "agent prompt frontmatter rejected",
			agentPath:   "agent.md",
			agent:       "---\nmodel: model-fast\nprompt: legacy.md\n---\nbody",
			wantMessage: "prompt frontmatter is not supported",
		},
		{
			name:        "seed is not a body field",
			agentPath:   "agent.md",
			agent:       "---\nmodel: model-fast\nseed: true\n---\nbody",
			wantMessage: `unknown field "seed"`,
		},
		{
			name:        "temperature is not a field an agent may pin",
			agentPath:   "agent.md",
			agent:       "---\nmodel: model-fast\ntemperature: 0\n---\nbody",
			wantMessage: `unknown field "temperature"`,
		},
		{
			name:        "legacy_thinking is not a body field",
			agentPath:   "agent.md",
			agent:       "---\nmodel: model-fast\nlegacy_thinking: true\n---\nbody",
			wantMessage: `unknown field "legacy_thinking"`,
		},
		{
			name:        "cache_ttl is not a body field",
			agentPath:   "agent.md",
			agent:       "---\nmodel: model-fast\ncache_ttl: 600\n---\nbody",
			wantMessage: `unknown field "cache_ttl"`,
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
			models := test.models
			if models == "" {
				models = validModelsYAML()
			}
			writeTestFile(t, filepath.Join(root, "models.yaml"), models)
			writeTestFile(t, filepath.Join(root, test.agentPath), test.agent)
			_, report := Load(root, DefaultModelsFile)
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

// A models.yaml that did not load defines no alias, so every agent naming one would
// be reported as naming a model that does not exist. The file's own diagnostic is the
// true one and it stands alone.
func TestBrokenModelsFileReportsOnlyItself(t *testing.T) {
	tests := []struct {
		name   string
		models string
		write  bool
	}{
		{name: "malformed YAML", models: "models: [", write: true},
		{
			name:   "alias defined twice",
			models: "models:\n  model-fast:\n    model: openai/one\n  model-fast:\n    model: openai/two\n",
			write:  true,
		},
		{name: "no models configured", models: "models:\n", write: true},
		{
			name:   "alias extends an alias the file does not define",
			models: "models:\n  model-fast:\n    extends: missing\n",
			write:  true,
		},
		{name: "no models file at all"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if test.write {
				writeTestFile(t, filepath.Join(root, "models.yaml"), test.models)
			}
			writeTestFile(t, filepath.Join(root, "agent.md"), agentFile("model-fast", "prompt"))
			_, report := Load(root, DefaultModelsFile)
			if hasDiagnostic(report, SeverityError, "unknown model alias") {
				t.Fatalf("a defined alias reported unknown: %v", report.Diagnostics)
			}
			if report.ErrorCount() != 1 || report.Diagnostics[0].Path != "models.yaml" {
				t.Fatalf("want one models.yaml error: %v", report.Diagnostics)
			}
		})
	}
}

func TestModelsFileSelection(t *testing.T) {
	outsideTable := filepath.Join(t.TempDir(), "elsewhere.yaml")
	tests := []struct {
		name        string
		modelsFile  string
		wantModel   string
		wantMessage string
	}{
		{name: "default", modelsFile: DefaultModelsFile, wantModel: "openai/upstream-fast"},
		{name: "alternate table", modelsFile: "models.alt.yaml", wantModel: "alt/upstream-fast"},
		{name: "absent table", modelsFile: "models.absent.yaml", wantMessage: "no such file"},
		{name: "table outside the config directory", modelsFile: outsideTable, wantModel: "outside/upstream-fast"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := writeValidConfig(t)
			writeTestFile(t, filepath.Join(root, "models.alt.yaml"),
				strings.ReplaceAll(validModelsYAML(), "openai/", "alt/"))
			writeTestFile(t, outsideTable, strings.ReplaceAll(validModelsYAML(), "openai/", "outside/"))

			registry, report := Load(root, test.modelsFile)
			if test.wantMessage == "" {
				if report.HasErrors() {
					t.Fatalf("load errors: %v", report.Diagnostics)
				}
				if model := registry.Configs["simple"].Model; model != test.wantModel {
					t.Errorf("simple model = %q, want %q", model, test.wantModel)
				}
				return
			}
			if !hasDiagnostic(report, SeverityError, test.wantMessage) {
				t.Fatalf("missing diagnostic %q: %v", test.wantMessage, report.Diagnostics)
			}
			if !hasDiagnosticPath(report, test.modelsFile) {
				t.Errorf("diagnostic does not name %q: %v", test.modelsFile, report.Diagnostics)
			}
		})
	}
}

// A diagnostic that names a Go type tells a config author about chancery rather than
// about their file, so unknown fields are reported by name and nothing else.
func TestUnknownFieldLeaksNoGoType(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "models.yaml"), validModelsYAML())
	writeTestFile(t, filepath.Join(root, "agent.md"), "---\nmodel: model-fast\nseed: true\n---\nbody")
	_, report := Load(root, DefaultModelsFile)
	for _, diagnostic := range report.Diagnostics {
		for _, leak := range []string{"agentEntry", "agentFrontmatter", "modelEntry", "prompts."} {
			if strings.Contains(diagnostic.Message, leak) {
				t.Errorf("diagnostic %q names %q", diagnostic.Message, leak)
			}
		}
	}
}

func TestResolveAgent(t *testing.T) {
	registry, report := Load(writeValidConfig(t), DefaultModelsFile)
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
		{name: "top-level agent", reference: "simple", wantPath: "simple", wantModel: "openai/upstream-fast"},
		{name: "folder index default", reference: "folder", wantPath: "folder", wantName: "fast", wantModel: "openai/upstream-fast"},
		{name: "explicit named model", reference: "folder.deep", wantPath: "folder", wantName: "deep", wantModel: "openai/upstream-deep"},
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
	_, report := Load(root, DefaultModelsFile)
	if !hasDiagnostic(report, SeverityError, "collides with named model route") {
		t.Fatalf("missing collision diagnostic: %v", report.Diagnostics)
	}
}

// tools/ is reserved and inert: a file under it is neither routed nor read, so it is
// not an orphan either.
func TestToolsDirectoryAnswersNoRoute(t *testing.T) {
	root := writeValidConfig(t)
	writeTestFile(t, filepath.Join(root, "tools", "shell", "grep.run_shell.md"), "tool prompt")
	registry, report := Load(root, DefaultModelsFile)
	if report.HasErrors() || report.WarningCount() != 0 {
		t.Fatalf("tools/ produced diagnostics: %v", report.Diagnostics)
	}
	if slices.Contains(registry.AgentPaths(), "tools/shell/grep.run_shell") {
		t.Fatalf("tools/ answered a route: %v", registry.AgentPaths())
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
	_, report := Load(root, DefaultModelsFile)
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
	_, report := Load(root, DefaultModelsFile)
	if !hasDiagnostic(report, SeverityError, "escapes config directory through symlink") {
		t.Fatalf("missing tools directory diagnostic: %v", report.Diagnostics)
	}
}

func TestResolveModel(t *testing.T) {
	models := map[string]modelEntry{
		"base":     {Model: "openai/upstream", Verbosity: "low"},
		"child":    {Extends: "base", ServiceTier: "priority"},
		"renamed":  {Extends: "base", Model: "openai/other"},
		"cyclic":   {Extends: "cyclic"},
		"orphaned": {Extends: "missing"},
	}
	tests := []struct {
		name      string
		key       string
		want      modelEntry
		wantError bool
	}{
		{
			name: "child inherits the whole parent",
			key:  "child",
			want: modelEntry{Model: "openai/upstream", Verbosity: "low", ServiceTier: "priority"},
		},
		{
			name: "child overrides the model",
			key:  "renamed",
			want: modelEntry{Model: "openai/other", Verbosity: "low"},
		},
		{name: "extends cycle", key: "cyclic", wantError: true},
		{name: "extends unknown alias", key: "orphaned", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveModel(test.key, models)
			if test.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve model: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("model mismatch (-want +got):\n%s", diff)
			}
		})
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
		"models.yaml":        validModelsYAML(),
		"simple.md":          agentFile("model-fast", "simple prompt"),
		"prio.md":            agentFile("model-fast-prio", "prio prompt"),
		"folder/index.md":    "---\ndescription: named agent\nmodels:\n  fast:\n    model: model-fast\n  deep:\n    model: model-deep\n    reasoning_effort: high\ndefault: fast\n---\n[fragment.md]\n\nfolder prompt",
		"folder/fragment.md": "local fragment",
	}
	for path, content := range files {
		writeTestFile(t, filepath.Join(root, path), content)
	}
	return root
}

func validModelsYAML() string {
	return `models:
  model-fast:
    model: openai/upstream-fast
    reasoning_effort: low
  model-fast-prio:
    extends: model-fast
    service_tier: priority
  model-deep:
    model: openai/upstream-deep
    reasoning_effort: medium
`
}

func agentFile(model, body string) string {
	return fmt.Sprintf("---\ndescription: test agent\nmodel: %s\n---\n%s", model, body)
}

func hasDiagnosticPath(report Report, path string) bool {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Path == path {
			return true
		}
	}
	return false
}

func hasDiagnostic(report Report, severity Severity, message string) bool {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Severity == severity && strings.Contains(diagnostic.Message, message) {
			return true
		}
	}
	return false
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
