package cli

import (
	"os"
	"strings"
	"testing"
)

func TestRunCallValidatesArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing agent", args: []string{"--config", "unused", "call"}},
		{name: "extra agent", args: []string{"--config", "unused", "call", "one", "two"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, _, stderr := executeCLI(test.args)
			if code != exitFailure {
				t.Errorf("exit code = %d, want %d", code, exitFailure)
			}
			if strings.Count(stderr, "accepts 1 arg(s)") != 1 {
				t.Errorf("stderr argument error count = %d, want 1: %q", strings.Count(stderr, "accepts 1 arg(s)"), stderr)
			}
		})
	}
}

func TestReadInput(t *testing.T) {
	tests := []struct {
		name  string
		value string
		stdin string
		want  string
	}{
		{name: "literal", value: "from flag", want: "from flag"},
		{name: "stdin", stdin: "from stdin", want: "from stdin"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := readInput(test.value, strings.NewReader(test.stdin))
			if err != nil {
				t.Fatalf("read input: %v", err)
			}
			if got != test.want {
				t.Errorf("input = %q, want %q", got, test.want)
			}
		})
	}
}

func TestReadInputFile(t *testing.T) {
	path := t.TempDir() + "/input.txt"
	if err := os.WriteFile(path, []byte("from file"), 0600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	got, err := readInput("@"+path, strings.NewReader(""))
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	if got != "from file" {
		t.Errorf("input = %q", got)
	}
}
