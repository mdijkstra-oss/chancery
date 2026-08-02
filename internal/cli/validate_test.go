package cli

import (
	"strings"
	"testing"
)

func TestRunValidateAcceptsValidConfig(t *testing.T) {
	code, stdout, stderr := executeCLI([]string{"--config", validConfig(t), "validate"})
	if code != exitSuccess {
		t.Fatalf("exit code = %d, want %d: %q", code, exitSuccess, stderr)
	}
	if !strings.Contains(stdout, "✓ config valid (0 warnings)") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestRunValidateReportsDiagnostics(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		wantCode    int
		wantSymbol  string
		wantPath    string
		wantMessage string
	}{
		{
			name: "an agent naming an alias models.yaml does not define",
			files: map[string]string{
				"models.yaml": validModels,
				"plain.md":    "---\ndescription: plain\nmodel: smart-prio\n---\nbody",
			},
			wantCode:    exitFailure,
			wantSymbol:  "✗",
			wantPath:    "plain.md",
			wantMessage: `unknown model alias "smart-prio"`,
		},
		{
			name: "one alias defined twice",
			files: map[string]string{
				"models.yaml": "models:\n  fast:\n    model: openai/one\n" +
					"  fast:\n    model: openai/two\n",
				"plain.md": "---\ndescription: plain\nmodel: fast\n---\nbody",
			},
			wantCode:    exitFailure,
			wantSymbol:  "✗",
			wantPath:    "models.yaml",
			wantMessage: `mapping key "fast" already defined`,
		},
		{
			name: "a frontmatter field the format has no position for",
			files: map[string]string{
				"models.yaml": validModels,
				"plain.md":    "---\ndescription: plain\nmodel: fast\nseed: true\n---\nbody",
			},
			wantCode:    exitFailure,
			wantSymbol:  "✗",
			wantPath:    "plain.md",
			wantMessage: `unknown field "seed"`,
		},
		{
			name: "a fragment no agent includes",
			files: map[string]string{
				"models.yaml": validModels,
				"plain.md":    "---\ndescription: plain\nmodel: fast\n---\nbody",
				"stray.md":    "a fragment nobody asked for",
			},
			wantCode:    exitFailure,
			wantSymbol:  "✗",
			wantPath:    "stray.md",
			wantMessage: "orphaned Markdown file",
		},
		{
			name: "an agent with nothing in its body",
			files: map[string]string{
				"models.yaml": validModels,
				"plain.md":    "---\ndescription: plain\nmodel: fast\n---\n",
			},
			wantCode:    exitSuccess,
			wantSymbol:  "!",
			wantPath:    "plain.md",
			wantMessage: "empty prompt body",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := writeConfigDir(t, test.files)
			code, stdout, stderr := executeCLI([]string{"--config", root, "validate"})
			if code != test.wantCode {
				t.Fatalf("exit code = %d, want %d: %q %q", code, test.wantCode, stdout, stderr)
			}
			line := test.wantSymbol + " " + test.wantPath + ": "
			if !strings.Contains(stdout, line) {
				t.Fatalf("stdout %q has no %q", stdout, line)
			}
			if !strings.Contains(stdout, test.wantMessage) {
				t.Fatalf("stdout %q does not report %q", stdout, test.wantMessage)
			}
			leaks := []string{"agentEntry", "agentFrontmatter", "modelEntry", "prompts."}
			for _, leak := range leaks {
				if strings.Contains(stdout, leak) {
					t.Errorf("stdout %q names the Go type %q", stdout, leak)
				}
			}
		})
	}
}

func TestRunValidateMissingDirectory(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "config before command", args: []string{"--config", "/path/that/does/not/exist", "validate"}},
		{name: "config after command", args: []string{"validate", "--config", "/path/that/does/not/exist"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := executeCLI(test.args)
			if code != exitFailure {
				t.Errorf("exit code = %d, want %d", code, exitFailure)
			}
			if !strings.Contains(stdout, "does/not/exist") {
				t.Errorf("stdout = %q", stdout)
			}
			if strings.Count(stderr, "config invalid") != 1 {
				t.Errorf("stderr validation error count = %d, want 1: %q", strings.Count(stderr, "config invalid"), stderr)
			}
			if strings.Contains(stderr, "Usage:") {
				t.Errorf("stderr includes usage for runtime error: %q", stderr)
			}
		})
	}
}
