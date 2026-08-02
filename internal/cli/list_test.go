package cli

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Every route and every alias behind it has to appear, in both renderings: list is
// how an operator finds out what the directory compiled to.
func TestRunListEnumeratesEveryRouteAndModel(t *testing.T) {
	root := validConfig(t)
	wantJSON := listOutput{
		Agents: []listAgent{
			{
				Path:        "named",
				Description: "named agent",
				Models: []listModel{
					{Name: "quick", Model: "openai/upstream-fast", ReasoningEffort: "low", Default: true},
					{Name: "thorough", Model: "openai/upstream-deep", ReasoningEffort: "high"},
				},
			},
			{Path: "plain", Description: "plain agent", Model: "openai/upstream-fast", Reasoning: "low"},
			{Path: "prio", Description: "prio agent", Model: "openai/upstream-fast", Reasoning: "low"},
		},
		Summary: listSummary{Agents: 3, Models: 4},
	}

	code, stdout, stderr := executeCLI([]string{"--config", root, "list", "--json"})
	if code != exitSuccess {
		t.Fatalf("exit code = %d: %q", code, stderr)
	}
	var decoded listOutput
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if !reflect.DeepEqual(decoded, wantJSON) {
		t.Fatalf("list --json = %#v\nwant %#v", decoded, wantJSON)
	}

	code, table, stderr := executeCLI([]string{"--config", root, "list"})
	if code != exitSuccess {
		t.Fatalf("exit code = %d: %q", code, stderr)
	}
	wantLines := []string{
		"named", ".quick (default)", ".thorough",
		"plain", "prio",
		"openai/upstream-fast", "openai/upstream-deep",
		"3 agents · 4 models",
	}
	for _, want := range wantLines {
		if !strings.Contains(table, want) {
			t.Errorf("list output %q has no %q", table, want)
		}
	}
}
