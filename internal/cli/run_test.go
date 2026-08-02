package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunParser(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantCode       int
		stdoutContains string
		stderrContains string
	}{
		{name: "root help", args: []string{"--help"}, wantCode: exitSuccess, stdoutContains: "Available Commands"},
		{name: "command help", args: []string{"call", "--help"}, wantCode: exitSuccess, stdoutContains: "call <agent-path>"},
		{name: "missing command", args: []string{"--config", "unused"}, wantCode: exitFailure, stderrContains: "a command is required"},
		{name: "unknown command", args: []string{"--config", "unused", "unknown"}, wantCode: exitFailure, stderrContains: "unknown command"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := executeCLI(test.args)
			if code != test.wantCode {
				t.Errorf("exit code = %d, want %d", code, test.wantCode)
			}
			if test.stdoutContains != "" && !strings.Contains(stdout, test.stdoutContains) {
				t.Errorf("stdout = %q", stdout)
			}
			if test.stderrContains != "" && strings.Count(stderr, test.stderrContains) != 1 {
				t.Errorf("stderr error count = %d, want 1: %q", strings.Count(stderr, test.stderrContains), stderr)
			}
		})
	}
}

// A command run with no --config reads ./config, and a directory that is not there
// is named rather than being served as no routes at all.
func TestRunFallsBackToTheDefaultConfigDir(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "serve", args: []string{"serve"}},
		{name: "validate", args: []string{"validate"}},
		{name: "list", args: []string{"list"}},
		{name: "call", args: []string{"call", "agent"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			code, stdout, stderr := executeCLI(test.args)
			if code != exitFailure {
				t.Errorf("exit code = %d, want %d", code, exitFailure)
			}
			if !strings.Contains(stdout+stderr, defaultConfigDir) {
				t.Errorf("output never names %q: %q", defaultConfigDir, stdout+stderr)
			}
		})
	}
}

// The directory is read from the working directory, so the same command in a
// configuration directory's parent needs no flag at all.
func TestRunReadsTheDefaultConfigDir(t *testing.T) {
	t.Chdir(t.TempDir())
	writeConfigDirAt(t, defaultConfigDir, map[string]string{
		"models.yaml": validModels,
		"plain.md":    "---\ndescription: plain agent\nmodel: fast\n---\nYou are plain.",
	})

	code, stdout, stderr := executeCLI([]string{"validate"})
	if code != exitSuccess {
		t.Fatalf("exit code = %d, want %d: %s", code, exitSuccess, stderr)
	}
	if !strings.Contains(stdout, "config valid") {
		t.Errorf("stdout = %q", stdout)
	}
}

func executeCLI(args []string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), args, strings.NewReader(""), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}
