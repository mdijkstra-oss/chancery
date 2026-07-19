package cli

import "testing"

func TestRunServeRejectsArguments(t *testing.T) {
	code, _, _ := executeCLI([]string{"--config", "unused", "serve", "extra"})
	if code != exitFailure {
		t.Errorf("exit code = %d, want %d", code, exitFailure)
	}
}
