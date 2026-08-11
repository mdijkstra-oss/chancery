package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mdijkstra-oss/chancery/internal/prompts"
)

// writeConfigDir builds a real configuration directory. Every CLI test loads one of
// these rather than a hand-built registry, so what the commands read is what an
// operator would have written.
func writeConfigDir(t *testing.T, files map[string]string) string {
	t.Helper()
	return writeConfigDirAt(t, t.TempDir(), files)
}

// writeConfigDirAt builds one at a named path, for the tests that care where it sits.
func writeConfigDirAt(t *testing.T, root string, files map[string]string) string {
	t.Helper()
	for path, content := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("create directory: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return root
}

const validModels = "models:\n" +
	"  fast:\n    model: openai/upstream-fast\n    reasoning_effort: low\n" +
	"  fast-prio:\n    extends: fast\n    service_tier: priority\n" +
	"  deep:\n    model: openai/upstream-deep\n    reasoning_effort: high\n"

// validConfig is three routes over three aliases: a plain agent, an agent behind an
// inheriting alias, and one carrying two named models.
func validConfig(t *testing.T) string {
	t.Helper()
	return writeConfigDir(t, map[string]string{
		"models.yaml": validModels,
		"plain.md":    "---\ndescription: plain agent\nmodel: fast\n---\nYou are plain.",
		"prio.md":     "---\ndescription: prio agent\nmodel: fast-prio\n---\nYou are prio.",
		"named/index.md": "---\ndescription: named agent\nmodels:\n" +
			"  quick:\n    model: fast\n  thorough:\n    model: deep\ndefault: quick\n---\n" +
			"You are named.",
	})
}

func TestLoadRegistry(t *testing.T) {
	registry, err := loadRegistry(configLocation{Root: validConfig(t), ModelsFile: prompts.DefaultModelsFile})
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	want := []string{"named", "plain", "prio"}
	got := registry.AgentPaths()
	if len(got) != len(want) {
		t.Fatalf("agent paths = %v, want %v", got, want)
	}
	for index, path := range want {
		if got[index] != path {
			t.Fatalf("agent paths = %v, want %v", got, want)
		}
	}
	if model := registry.Configs["prio"].Model; model != "openai/upstream-fast" {
		t.Errorf("prio model = %q, want openai/upstream-fast", model)
	}
	if tier := registry.Configs["prio"].ServiceTier; tier != "priority" {
		t.Errorf("prio service tier = %q, want priority", tier)
	}
}

// An error report becomes one message per failing file, so the operator is told
// everything wrong with the directory rather than the first thing found.
func TestLoadRegistryReportsEveryError(t *testing.T) {
	root := writeConfigDir(t, map[string]string{
		"models.yaml": validModels,
		"first.md":    "---\ndescription: first\nmodel: missing-alias\n---\nbody",
		"second.md":   "---\ndescription: second\nmodel: other-missing\n---\nbody",
	})
	_, err := loadRegistry(configLocation{Root: root, ModelsFile: prompts.DefaultModelsFile})
	if err == nil {
		t.Fatal("want an error, got none")
	}
	for _, want := range []string{"first.md", "second.md", "missing-alias", "other-missing"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
	if lines := strings.Count(err.Error(), "✗"); lines != 2 {
		t.Errorf("error lines = %d, want 2: %q", lines, err)
	}
}
