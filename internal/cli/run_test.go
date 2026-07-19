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

func TestRunRequiresConfig(t *testing.T) {
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
			code, _, stderr := executeCLI(test.args)
			if code != exitFailure {
				t.Errorf("exit code = %d, want %d", code, exitFailure)
			}
			errorText := "required flag(s) \"config\" not set"
			if strings.Count(stderr, errorText) != 1 {
				t.Errorf("stderr error count = %d, want 1: %q", strings.Count(stderr, errorText), stderr)
			}
		})
	}
}

func executeCLI(args []string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), args, strings.NewReader(""), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}
