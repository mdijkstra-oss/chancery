package cli

import (
	"strings"
	"testing"
)

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
