package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/matthijn/hermes-logos/internal/prompts"

	"github.com/spf13/cobra"
)

type listModel struct {
	Name            string `json:"name,omitempty"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	Default         bool   `json:"default,omitempty"`
}

type listAgent struct {
	Path        string      `json:"path"`
	Description string      `json:"description,omitempty"`
	Model       string      `json:"model,omitempty"`
	Reasoning   string      `json:"reasoning_effort,omitempty"`
	Models      []listModel `json:"models,omitempty"`
}

type listSummary struct {
	Agents    int `json:"agents"`
	Models    int `json:"models"`
	Providers int `json:"providers"`
}

type listOutput struct {
	Agents  []listAgent `json:"agents"`
	Summary listSummary `json:"summary"`
}

func newListCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "list",
		Short: "List configured agents and models",
		Args:  cobra.NoArgs,
		RunE:  runListCommand,
	}
	command.Flags().Bool("json", false, "print JSON")
	return command
}

func runListCommand(command *cobra.Command, _ []string) error {
	configPath, err := commandConfigPath(command)
	if err != nil {
		return err
	}
	asJSON, err := command.Flags().GetBool("json")
	if err != nil {
		return fmt.Errorf("read JSON flag: %w", err)
	}
	registry, err := loadRegistry(configPath)
	if err != nil {
		return err
	}
	return runList(registry, asJSON, command.OutOrStdout())
}

func runList(registry prompts.Registry, asJSON bool, output io.Writer) error {
	list := buildListOutput(registry)
	if asJSON {
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(list); err != nil {
			return fmt.Errorf("encode list: %w", err)
		}
		return nil
	}
	return writeListTable(list, output)
}

func buildListOutput(registry prompts.Registry) listOutput {
	agents := make([]listAgent, 0, len(registry.Agents))
	for _, path := range registry.AgentPaths() {
		agent := listAgent{Path: path, Description: registry.Descriptions[path]}
		if named := registry.NamedConfigs[path]; len(named) > 0 {
			names := slices.Sorted(maps.Keys(named))
			for _, name := range names {
				cfg := named[name]
				agent.Models = append(agent.Models, listModel{Name: name, Model: cfg.Model, ReasoningEffort: cfg.ReasoningEffort, Default: registry.Defaults[path] == name})
			}
		} else {
			cfg := registry.Configs[path]
			agent.Model = cfg.Model
			agent.Reasoning = cfg.ReasoningEffort
		}
		agents = append(agents, agent)
	}
	return listOutput{
		Agents: agents,
		Summary: listSummary{
			Agents:    len(registry.Agents),
			Models:    registry.ModelCount(),
			Providers: registry.ProviderCount(),
		},
	}
}

func renderListTable(output listOutput) string {
	var buf strings.Builder
	writer := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "PATH\tMODEL\tREASONING")
	for _, agent := range output.Agents {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", agent.Path, agent.Model, agent.Reasoning)
		for _, model := range agent.Models {
			marker := ""
			if model.Default {
				marker = " (default)"
			}
			fmt.Fprintf(writer, "  .%s%s\t%s\t%s\n", model.Name, marker, model.Model, model.ReasoningEffort)
		}
	}
	writer.Flush()
	fmt.Fprintf(&buf, "%d agents · %d models · %d providers\n", output.Summary.Agents, output.Summary.Models, output.Summary.Providers)
	return buf.String()
}

func writeListTable(output listOutput, destination io.Writer) error {
	if _, err := io.WriteString(destination, renderListTable(output)); err != nil {
		return fmt.Errorf("write list: %w", err)
	}
	return nil
}
