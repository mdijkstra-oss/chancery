package prompts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFolders(t *testing.T) {
	tmpDir := t.TempDir()

	cases := []struct {
		name     string
		folders  map[string]map[string]string
		load     []string
		expected string
		wantErr  bool
	}{
		{
			name: "single folder single file",
			folders: map[string]map[string]string{
				"base": {"00-intro.md": "intro"},
			},
			load:     []string{"base"},
			expected: "intro",
		},
		{
			name: "single folder multiple files sorted",
			folders: map[string]map[string]string{
				"base": {
					"02-second.md": "second",
					"00-first.md":  "first",
				},
			},
			load:     []string{"base"},
			expected: "first\n\nsecond",
		},
		{
			name: "multiple folders joined with separator",
			folders: map[string]map[string]string{
				"base": {"00-base.md": "base content"},
				"plan": {"00-plan.md": "plan content"},
			},
			load:     []string{"base", "plan"},
			expected: "base content\n\n---\n\nplan content",
		},
		{
			name: "multiple folders each with multiple files",
			folders: map[string]map[string]string{
				"base":    {"00-a.md": "a", "01-b.md": "b"},
				"execute": {"00-exec.md": "exec"},
			},
			load:     []string{"base", "execute"},
			expected: "a\n\nb\n\n---\n\nexec",
		},
		{
			name:     "missing folder returns error",
			folders:  map[string]map[string]string{},
			load:     []string{"nonexistent"},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tc.name)
			setupFolders(t, testDir, tc.folders)

			got, err := LoadFolders(testDir, tc.load)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.expected {
				t.Errorf("got %q, want %q", got, tc.expected)
			}
		})
	}
}

func setupFolders(t *testing.T, baseDir string, folders map[string]map[string]string) {
	t.Helper()
	for folder, files := range folders {
		dir := filepath.Join(baseDir, folder)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
		}
	}
}
