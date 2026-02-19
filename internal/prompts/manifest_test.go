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
			got, err := ResolveManifest(tc.lines, reader, shared, nil)
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
		got, err := ResolveManifest(lines, reader, shared, skipChat)
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
			path:      "/agents/expert/qualitative-researcher/index.md",
			agentsDir: "/agents",
			want:      "expert/qualitative-researcher",
		},
		{
			name:      "named file becomes parent/name",
			path:      "/agents/expert/qualitative-researcher/plan.md",
			agentsDir: "/agents",
			want:      "expert/qualitative-researcher/plan",
		},
		{
			name:      "top-level index",
			path:      "/agents/analyst/index.md",
			agentsDir: "/agents",
			want:      "analyst",
		},
		{
			name:      "deep path named file",
			path:      "/agents/expert/qualitative-researcher/compact.md",
			agentsDir: "/agents",
			want:      "expert/qualitative-researcher/compact",
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
	for name, content := range sharedFiles {
		writeTestFile(t, filepath.Join(promptsDir, name), content)
	}

	agentFiles := map[string]string{
		"agents/expert/qualitative-researcher/index.md": "[nabu/identity.md]\n[nabu/discipline.md]\n\nYou are an expert.",
		"agents/expert/qualitative-researcher/plan.md":  "[nabu/identity.md]\n\nPlan the work.\n\n[chat/style.md]",
		"agents/analyst/index.md":                       "[nabu/identity.md]\n\nYou analyze.",
	}
	for name, content := range agentFiles {
		writeTestFile(t, filepath.Join(promptsDir, name), content)
	}

	configFiles := map[string]string{
		"agents/expert/config.json":                          `{"model": "gpt-5.2", "reasoning_effort": "high"}`,
		"agents/expert/qualitative-researcher/config.json":   `{"model": "gpt-5.2", "reasoning_effort": "high", "verbosity": "low"}`,
		"agents/analyst/config.json":                         `{"model": "gpt-5.2", "reasoning_effort": "medium"}`,
	}
	for name, content := range configFiles {
		writeTestFile(t, filepath.Join(promptsDir, name), content)
	}

	registry := CompileRegistry(promptsDir)

	agentCases := []struct {
		key        string
		wantPrompt string
	}{
		{
			key:        "expert/qualitative-researcher",
			wantPrompt: "I am Nabu.\nBe disciplined.\n\nYou are an expert.",
		},
		{
			key:        "expert/qualitative-researcher/plan",
			wantPrompt: "I am Nabu.\n\nPlan the work.\n\nBe direct.",
		},
		{
			key:        "analyst",
			wantPrompt: "I am Nabu.\n\nYou analyze.",
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

	configCases := []struct {
		key  string
		want PromptConfig
	}{
		{
			key:  "expert/qualitative-researcher",
			want: PromptConfig{Model: "gpt-5.2", ReasoningEffort: "high", Verbosity: "low"},
		},
		{
			key:  "expert/qualitative-researcher/plan",
			want: PromptConfig{Model: "gpt-5.2", ReasoningEffort: "high", Verbosity: "low"},
		},
		{
			key:  "analyst",
			want: PromptConfig{Model: "gpt-5.2", ReasoningEffort: "medium"},
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
