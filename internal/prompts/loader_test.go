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
		name string
		opts ComposeOpts
		want string
	}{
		{
			name: "orchestrator gets base layer with chat and tools",
			opts: ComposeOpts{Folder: "nabu", Tools: []string{"orientate"}, Chat: true},
			want: "base-id\n\nbase-disc\n\nchat-style\n\norchestration-body",
		},
		{
			name: "expert gets base plus expert layer",
			opts: ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{"run_local_shell"}},
			want: "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id\n\nshell-body",
		},
		{
			name: "expert task walks up through ancestor layers",
			opts: ComposeOpts{Folder: "nabu/expert/researcher/apply-codebook", Tools: []string{}},
			want: "base-id\n\nbase-disc\n\nexpert-base\n\nresearcher-id\n\napply-task",
		},
		{
			name: "chat false excludes chat prompts",
			opts: ComposeOpts{Folder: "nabu", Tools: []string{}, Chat: false},
			want: "base-id\n\nbase-disc",
		},
		{
			name: "frontmatter filters out unavailable tools",
			opts: ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{"orientate"}},
			want: "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id\n\norchestration-body",
		},
		{
			name: "no tools excludes all frontmatter files",
			opts: ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{}},
			want: "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id",
		},
		{
			name: "chat comes after ancestors but before tools",
			opts: ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{"run_local_shell"}, Chat: true},
			want: "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id\n\nchat-style\n\nshell-body",
		},
		{
			name: "extra plan appended last",
			opts: ComposeOpts{Folder: "nabu", Tools: []string{"orientate"}, Chat: true, Extra: "plan"},
			want: "base-id\n\nbase-disc\n\nchat-style\n\norchestration-body\n\nplan-extra",
		},
		{
			name: "extra exec appended last",
			opts: ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{}, Extra: "exec"},
			want: "base-id\n\nbase-disc\n\nexpert-base\n\nanalyst-id\n\nexec-extra",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ComposePrompt(base, tc.opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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
		"nabu/01-identity.md":                              "base-id",
		"nabu/02-discipline.md":                            "base-disc",
		"tools/chat/style.md":                         "chat-style",
		"tools/orientation.md":                        "---\nrequires:\n  - orientate\n---\norchestration-body",
		"tools/shell.md":                              "---\nrequires:\n  - run_local_shell\n---\nshell-body",
		"tools/patching/patching.md":                  "---\nrequires:\n  - apply_local_patch\n---\npatching-body",
		"nabu/expert/01-identity.md":                       "expert-base",
		"nabu/expert/analyst/01-identity.md":               "analyst-id",
		"nabu/expert/researcher/01-identity.md":            "researcher-id",
		"nabu/expert/researcher/apply-codebook/01-task.md": "apply-task",
		"extra/plan/01-plan.md":                             "plan-extra",
		"extra/exec/01-exec.md":                             "exec-extra",
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
