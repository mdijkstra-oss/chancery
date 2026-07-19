package cli

import (
	"fmt"
	"io"

	"github.com/matthijn/hermes-logos/internal/prompts"

	"github.com/spf13/cobra"
)

func newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Args:  cobra.NoArgs,
		RunE:  runValidateCommand,
	}
}

func runValidateCommand(command *cobra.Command, _ []string) error {
	configPath, err := commandConfigPath(command)
	if err != nil {
		return err
	}
	return runValidate(configPath, command.OutOrStdout())
}

func runValidate(configPath string, output io.Writer) error {
	_, report := prompts.Load(configPath)
	for _, diagnostic := range report.Diagnostics {
		if _, err := fmt.Fprintf(output, "%s %s: %s\n", diagnosticSymbol(diagnostic), diagnostic.Path, diagnostic.Message); err != nil {
			return fmt.Errorf("write diagnostic: %w", err)
		}
	}
	if report.HasErrors() {
		return fmt.Errorf("config invalid (%d errors · %d warnings)", report.ErrorCount(), report.WarningCount())
	}
	if _, err := fmt.Fprintf(output, "✓ config valid (%d warnings)\n", report.WarningCount()); err != nil {
		return fmt.Errorf("write validation result: %w", err)
	}
	return nil
}

func diagnosticSymbol(diagnostic prompts.Diagnostic) string {
	if diagnostic.Severity == prompts.SeverityError {
		return "✗"
	}
	return "!"
}
