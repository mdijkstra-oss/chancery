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
			name: "converse gets shared chat agent tools",
			opts: ComposeOpts{Folder: "nabu/converse", Tools: []string{"orientate"}, Chat: true},
			want: "shared-id\n\nshared-disc\n\nchat-style\n\nconverse-experts\n\nconverse-coding\n\norchestration-body",
		},
		{
			name: "expert base gets shared plus expert layer",
			opts: ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{"run_local_shell"}},
			want: "shared-id\n\nshared-disc\n\nexpert-base\n\nanalyst-id\n\nshell-body",
		},
		{
			name: "expert task walks up through ancestor layers",
			opts: ComposeOpts{Folder: "nabu/expert/researcher/apply-codebook", Tools: []string{}},
			want: "shared-id\n\nshared-disc\n\nexpert-base\n\nresearcher-id\n\napply-task",
		},
		{
			name: "chat false excludes chat prompts",
			opts: ComposeOpts{Folder: "nabu/converse", Tools: []string{}, Chat: false},
			want: "shared-id\n\nshared-disc\n\nconverse-experts\n\nconverse-coding",
		},
		{
			name: "frontmatter filters out unavailable tools",
			opts: ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{"orientate"}},
			want: "shared-id\n\nshared-disc\n\nexpert-base\n\nanalyst-id\n\norchestration-body",
		},
		{
			name: "no tools excludes all frontmatter files",
			opts: ComposeOpts{Folder: "nabu/expert/analyst", Tools: []string{}},
			want: "shared-id\n\nshared-disc\n\nexpert-base\n\nanalyst-id",
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
	expertCfg := `{"model": "gpt-5", "reasoning_effort": "high"}`
	writeFile(t, filepath.Join(base, "nabu/expert/config.json"), expertCfg)
	os.MkdirAll(filepath.Join(base, "nabu/expert/researcher/apply-codebook"), 0755)

	cases := []struct {
		name    string
		folder  string
		want    PromptConfig
		wantErr bool
	}{
		{
			name:   "exact folder has config",
			folder: "nabu/expert",
			want:   PromptConfig{Model: "gpt-5", ReasoningEffort: "high"},
		},
		{
			name:   "walks up to parent config",
			folder: "nabu/expert/researcher/apply-codebook",
			want:   PromptConfig{Model: "gpt-5", ReasoningEffort: "high"},
		},
		{
			name:    "no config anywhere",
			folder:  "nabu/missing",
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
			folder: "converse",
			want:   []string{"converse"},
		},
		{
			name:   "two segments",
			folder: "nabu/converse",
			want:   []string{"nabu", "nabu/converse"},
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
		"nabu/shared/01-identity.md":                         "shared-id",
		"nabu/shared/02-discipline.md":                       "shared-disc",
		"nabu/tools/chat/style.md":                           "chat-style",
		"nabu/tools/orchestration.md":                        "---\nrequires:\n  - orientate\n---\norchestration-body",
		"nabu/tools/shell.md":                                "---\nrequires:\n  - run_local_shell\n---\nshell-body",
		"nabu/tools/patching/patching.md":                    "---\nrequires:\n  - apply_local_patch\n---\npatching-body",
		"nabu/converse/01-experts.md":                        "converse-experts",
		"nabu/converse/02-coding.md":                         "converse-coding",
		"nabu/expert/01-identity.md":                         "expert-base",
		"nabu/expert/analyst/01-identity.md":                 "analyst-id",
		"nabu/expert/researcher/01-identity.md":              "researcher-id",
		"nabu/expert/researcher/apply-codebook/01-task.md":   "apply-task",
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
