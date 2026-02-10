package prompts

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseFrontmatter(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantFM    Frontmatter
		wantBody  string
	}{
		{
			name:     "no frontmatter",
			input:    "just content",
			wantFM:   Frontmatter{},
			wantBody: "just content",
		},
		{
			name:     "single require",
			input:    "---\nrequires:\n  - run_local_shell\n---\nbody here",
			wantFM:   Frontmatter{Requires: []string{"run_local_shell"}},
			wantBody: "body here",
		},
		{
			name:     "multiple requires",
			input:    "---\nrequires:\n  - orientate\n  - run_local_shell\n---\ncontent",
			wantFM:   Frontmatter{Requires: []string{"orientate", "run_local_shell"}},
			wantBody: "content",
		},
		{
			name:     "unclosed frontmatter",
			input:    "---\nrequires:\n  - foo\nno closing",
			wantFM:   Frontmatter{},
			wantBody: "---\nrequires:\n  - foo\nno closing",
		},
		{
			name:     "empty frontmatter",
			input:    "---\n---\nbody",
			wantFM:   Frontmatter{},
			wantBody: "body",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotFM, gotBody := ParseFrontmatter(tc.input)
			if diff := cmp.Diff(tc.wantFM, gotFM); diff != "" {
				t.Errorf("frontmatter mismatch (-want +got):\n%s", diff)
			}
			if gotBody != tc.wantBody {
				t.Errorf("body: got %q, want %q", gotBody, tc.wantBody)
			}
		})
	}
}

func TestHasRequired(t *testing.T) {
	cases := []struct {
		name      string
		requires  []string
		available []string
		want      bool
	}{
		{
			name:      "empty requires always true",
			requires:  nil,
			available: []string{"foo"},
			want:      true,
		},
		{
			name:      "single match",
			requires:  []string{"foo"},
			available: []string{"foo", "bar"},
			want:      true,
		},
		{
			name:      "no match",
			requires:  []string{"baz"},
			available: []string{"foo", "bar"},
			want:      false,
		},
		{
			name:      "or semantics any match sufficient",
			requires:  []string{"baz", "foo"},
			available: []string{"foo"},
			want:      true,
		},
		{
			name:      "empty available with requires",
			requires:  []string{"foo"},
			available: nil,
			want:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := HasRequired(tc.requires, tc.available)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
