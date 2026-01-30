package prompts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFolder(t *testing.T) {
	tmpDir := t.TempDir()

	cases := []struct {
		name     string
		files    map[string]string
		expected string
		wantErr  bool
	}{
		{
			name:     "single file",
			files:    map[string]string{"00-intro.md": "intro"},
			expected: "intro",
		},
		{
			name: "multiple files sorted",
			files: map[string]string{
				"02-second.md": "second",
				"00-first.md":  "first",
			},
			expected: "first\n\nsecond",
		},
		{
			name: "subdirectory recursion",
			files: map[string]string{
				"01-intro.md":         "intro",
				"02-skills/01-a.md":   "skill-a",
				"02-skills/02-b.md":   "skill-b",
				"03-outro.md":         "outro",
			},
			expected: "intro\n\nskill-a\n\nskill-b\n\noutro",
		},
		{
			name:     "empty folder returns error",
			files:    map[string]string{},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tc.name)
			setupFolder(t, testDir, tc.files)

			got, err := LoadFolder(tmpDir, tc.name)

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

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cases := []struct {
		name     string
		json     string
		expected PromptConfig
		wantErr  bool
	}{
		{
			name: "full config",
			json: `{"model": "gpt-4", "reasoning_effort": "high", "verbosity": "low"}`,
			expected: PromptConfig{
				Model:           "gpt-4",
				ReasoningEffort: "high",
				Verbosity:       "low",
			},
		},
		{
			name:     "empty config",
			json:     `{}`,
			expected: PromptConfig{},
		},
		{
			name:     "missing file",
			json:     "",
			expected: PromptConfig{},
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			folder := filepath.Join(tmpDir, tc.name)
			if err := os.MkdirAll(folder, 0755); err != nil {
				t.Fatalf("failed to create dir: %v", err)
			}

			if tc.json != "" {
				if err := os.WriteFile(filepath.Join(folder, "config.json"), []byte(tc.json), 0644); err != nil {
					t.Fatalf("failed to write config: %v", err)
				}
			}

			got, err := LoadConfig(tmpDir, tc.name)

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
				t.Errorf("got %+v, want %+v", got, tc.expected)
			}
		})
	}
}

func setupFolder(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create parent dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}
}
