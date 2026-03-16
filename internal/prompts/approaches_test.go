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

func TestIsIndexKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"index", true},
		{"qual-coding/project/index", true},
		{"qual-coding/codebook/review", false},
		{"indexing", false},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			if got := IsIndexKey(tc.key); got != tc.want {
				t.Errorf("IsIndexKey(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestParentIndexKeys(t *testing.T) {
	cases := []struct {
		key  string
		want []string
	}{
		{"simple", nil},
		{"a/b", []string{"a/index"}},
		{"a/b/c", []string{"a/index", "a/b/index"}},
		{"a/b/c/d", []string{"a/index", "a/b/index", "a/b/c/index"}},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			got := ParentIndexKeys(tc.key)
			if len(got) == 0 {
				got = nil
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectIndexKeys(t *testing.T) {
	cases := []struct {
		name string
		keys []string
		want []string
	}{
		{
			name: "single deep key",
			keys: []string{"a/b/c"},
			want: []string{"a/b/index", "a/index", "index"},
		},
		{
			name: "multiple keys shared parents",
			keys: []string{"a/b/c", "a/b/d"},
			want: []string{"a/b/index", "a/index", "index"},
		},
		{
			name: "different branches",
			keys: []string{"a/b", "c/d"},
			want: []string{"a/index", "c/index", "index"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CollectIndexKeys(tc.keys)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveApproachKeys(t *testing.T) {
	cases := []struct {
		name      string
		requested []string
		want      []string
	}{
		{
			name:      "single key includes indexes",
			requested: []string{"a/b/c"},
			want:      []string{"a/b/index", "a/index", "index", "a/b/c"},
		},
		{
			name:      "multiple keys deduped",
			requested: []string{"a/b/c", "a/b/d"},
			want:      []string{"a/b/index", "a/index", "index", "a/b/c", "a/b/d"},
		},
		{
			name:      "requested key is also an index",
			requested: []string{"a/b/index", "a/b/c"},
			want:      []string{"a/b/index", "a/index", "index", "a/b/c"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveApproachKeys(tc.requested)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
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

	writeFile(t, approachesDir, "index.md", "## Root index\n")
	writeFile(t, approachesDir, "topic/index.md", "## Topic index\n")
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

	if _, ok := reg.Entries["index"]; !ok {
		t.Error("expected root index in Entries")
	}
	if _, ok := reg.Entries["topic/index"]; !ok {
		t.Error("expected topic/index in Entries")
	}
	if _, ok := reg.Entries["topic/leaf"]; !ok {
		t.Error("expected topic/leaf in Entries")
	}

	leaf := reg.Entries["topic/leaf"]
	if leaf.Description != "A leaf approach." {
		t.Errorf("leaf description = %q, want %q", leaf.Description, "A leaf approach.")
	}
}
