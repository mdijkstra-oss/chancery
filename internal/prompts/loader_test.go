package prompts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestComposePrompt(t *testing.T) {
	base := setupPromptTree(t)

	cases := []struct {
		name        string
		opts        ComposeOpts
		wantPrompt  string
		wantSources []string
	}{
		{
			name:        "orchestrator gets base layer with chat and tools",
			opts:        ComposeOpts{Folder: "nabu", Tools: []string{"orientate"}, Chat: true},
			wantPrompt:  "base-id\n\nbase-disc\n\nchat-style\n\norchestration-body",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md", "tools/chat/style.md", "tools/orientation.orientate.md"},
		},
		{
			name:        "expert gets base plus expert layer",
			opts:        ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{"run_local_shell"}},
			wantPrompt:  "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id\n\nshell-body",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md", "nabu/expert/01-identity.md", "nabu/expert/analyst/01-identity.md", "tools/shell.run_local_shell.md"},
		},
		{
			name:        "ancestor walk excludes tool-gated domain files without tool",
			opts:        ComposeOpts{Folder: "nabu/expert/researcher/apply-codebook", Tools: []string{}},
			wantPrompt:  "base-id\n\nbase-disc\n\nexpert-base\n\nresearcher-id\n\ncoding-concepts\n\napply-task",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md", "nabu/expert/01-identity.md", "nabu/expert/researcher/01-identity.md", "nabu/expert/researcher/02-coding.md", "nabu/expert/researcher/apply-codebook/01-task.md"},
		},
		{
			name:        "ancestor walk includes tool-gated domain files with tool",
			opts:        ComposeOpts{Folder: "nabu/expert/researcher/apply-codebook", Tools: []string{"patch_json_block"}},
			wantPrompt:  "base-id\n\nbase-disc\n\nexpert-base\n\nresearcher-id\n\ncoding-concepts\n\ncoding-mechanics\n\napply-task",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md", "nabu/expert/01-identity.md", "nabu/expert/researcher/01-identity.md", "nabu/expert/researcher/02-coding.md", "nabu/expert/researcher/02-coding.patch_json_block.md", "nabu/expert/researcher/apply-codebook/01-task.md"},
		},
		{
			name:        "chat false excludes chat prompts",
			opts:        ComposeOpts{Folder: "nabu", Tools: []string{}, Chat: false},
			wantPrompt:  "base-id\n\nbase-disc",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md"},
		},
		{
			name:        "filename filters out unavailable tools",
			opts:        ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{"orientate"}},
			wantPrompt:  "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id\n\norchestration-body",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md", "nabu/expert/01-identity.md", "nabu/expert/analyst/01-identity.md", "tools/orientation.orientate.md"},
		},
		{
			name:        "no tools excludes all tool-gated files",
			opts:        ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{}},
			wantPrompt:  "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md", "nabu/expert/01-identity.md", "nabu/expert/analyst/01-identity.md"},
		},
		{
			name:        "chat comes after ancestors but before tools",
			opts:        ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{"run_local_shell"}, Chat: true},
			wantPrompt:  "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id\n\nchat-style\n\nshell-body",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md", "nabu/expert/01-identity.md", "nabu/expert/analyst/01-identity.md", "tools/chat/style.md", "tools/shell.run_local_shell.md"},
		},
		{
			name:        "extra plan appended last",
			opts:        ComposeOpts{Folder: "nabu", Tools: []string{"orientate"}, Chat: true, Extra: "plan"},
			wantPrompt:  "base-id\n\nbase-disc\n\nchat-style\n\norchestration-body\n\nplan-extra",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md", "tools/chat/style.md", "tools/orientation.orientate.md", "extra/plan/01-plan.md"},
		},
		{
			name:        "extra exec appended last",
			opts:        ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{}, Extra: "exec"},
			wantPrompt:  "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id\n\nexec-extra",
			wantSources: []string{"nabu/01-identity.md", "nabu/02-discipline.md", "nabu/expert/01-identity.md", "nabu/expert/analyst/01-identity.md", "extra/exec/01-exec.md"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ComposePrompt(base, tc.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantPrompt, got.Prompt); diff != "" {
				t.Errorf("prompt mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantSources, got.Sources); diff != "" {
				t.Errorf("sources mismatch (-want +got):\n%s", diff)
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

func TestResolveConfig(t *testing.T) {
	base := t.TempDir()
	baseCfg := `{"model": "gpt-5.2", "reasoning_effort": "medium"}`
	expertCfg := `{"model": "gpt-5.2", "reasoning_effort": "high"}`
	writeFile(t, filepath.Join(base, "nabu/config.json"), baseCfg)
	writeFile(t, filepath.Join(base, "nabu/expert/config.json"), expertCfg)
	os.MkdirAll(filepath.Join(base, "nabu/expert/researcher/apply-codebook"), 0755)

	cases := []struct {
		name    string
		folder  string
		want    PromptConfig
		wantErr bool
	}{
		{
			name:   "orchestrator gets base config",
			folder: "nabu",
			want:   PromptConfig{Model: "gpt-5.2", ReasoningEffort: "medium"},
		},
		{
			name:   "expert gets expert config",
			folder: "nabu/expert",
			want:   PromptConfig{Model: "gpt-5.2", ReasoningEffort: "high"},
		},
		{
			name:   "deep path walks up to expert config",
			folder: "nabu/expert/researcher/apply-codebook",
			want:   PromptConfig{Model: "gpt-5.2", ReasoningEffort: "high"},
		},
		{
			name:    "no config anywhere",
			folder:  "missing",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveConfig(base, tc.folder)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAncestorFolders(t *testing.T) {
	cases := []struct {
		name   string
		folder string
		want   []string
	}{
		{
			name:   "single segment",
			folder: "nabu",
			want:   []string{"nabu"},
		},
		{
			name:   "two segments",
			folder: "nabu/expert",
			want:   []string{"nabu", "nabu/expert"},
		},
		{
			name:   "deep path",
			folder: "nabu/expert/researcher/apply-codebook",
			want:   []string{"nabu", "nabu/expert", "nabu/expert/researcher", "nabu/expert/researcher/apply-codebook"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ancestorFolders(tc.folder)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func setupPromptTree(t *testing.T) string {
	t.Helper()
	base := t.TempDir()

	files := map[string]string{
		"nabu/01-identity.md":                                          "base-id",
		"nabu/02-discipline.md":                                        "base-disc",
		"tools/chat/style.md":                                          "chat-style",
		"tools/orientation.orientate.md":                                "orchestration-body",
		"tools/shell.run_local_shell.md":                                "shell-body",
		"tools/patching/patching.apply_local_patch.md":                  "patching-body",
		"nabu/expert/01-identity.md":                                    "expert-base",
		"nabu/expert/analyst/01-identity.md":                            "analyst-id",
		"nabu/expert/researcher/01-identity.md":                         "researcher-id",
		"nabu/expert/researcher/02-coding.md":                           "coding-concepts",
		"nabu/expert/researcher/02-coding.patch_json_block.md":          "coding-mechanics",
		"nabu/expert/researcher/apply-codebook/01-task.md":              "apply-task",
		"extra/plan/01-plan.md":                                         "plan-extra",
		"extra/exec/01-exec.md":                                         "exec-extra",
	}

	for name, content := range files {
		writeFile(t, filepath.Join(base, name), content)
	}
	return base
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}
