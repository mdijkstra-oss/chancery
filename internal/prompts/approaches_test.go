package prompts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestKeyFromPath(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"index.md", "index"},
		{"qual-coding/codebook/review.md", "qual-coding/codebook/review"},
		{"qual-coding/project/index.md", "qual-coding/project/index"},
		{"simple.md", "simple"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if got := KeyFromPath(tc.input); got != tc.want {
				t.Errorf("KeyFromPath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestExtractDescription(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "with frontmatter",
			content: "---\ndescription: There is material to code.\n---\n\n## Heading",
			want:    "There is material to code.",
		},
		{
			name:    "no frontmatter",
			content: "## Heading\n\nSome content",
			want:    "",
		},
		{
			name:    "frontmatter without description",
			content: "---\ntitle: Something\n---\n\nContent",
			want:    "",
		},
		{
			name:    "unclosed frontmatter",
			content: "---\ndescription: Oops",
			want:    "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractDescription(tc.content); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCompileApproaches(t *testing.T) {
	dir := t.TempDir()
	approachesDir := filepath.Join(dir, "modes", "approaches")

	writeFile(t, approachesDir, "topic/leaf.md", "---\ndescription: A leaf approach.\n---\n\n## Leaf\n")
	writeFile(t, approachesDir, "topic/nodesc.md", "## No description\n")

	reg := compileApproaches(dir)

	wantKeys := []string{"topic/leaf", "topic/nodesc"}
	if diff := cmp.Diff(wantKeys, reg.Keys); diff != "" {
		t.Errorf("Keys mismatch (-want +got):\n%s", diff)
	}

	wantDescriptions := map[string]string{"topic/leaf": "A leaf approach."}
	if diff := cmp.Diff(wantDescriptions, reg.Descriptions); diff != "" {
		t.Errorf("Descriptions mismatch (-want +got):\n%s", diff)
	}

	if _, ok := reg.Entries["topic/leaf"]; !ok {
		t.Error("expected topic/leaf in Entries")
	}

	leaf := reg.Entries["topic/leaf"]
	if leaf.Description != "A leaf approach." {
		t.Errorf("leaf description = %q, want %q", leaf.Description, "A leaf approach.")
	}
}
