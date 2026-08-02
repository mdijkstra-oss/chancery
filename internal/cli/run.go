package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

const (
	exitSuccess = 0
	exitFailure = 1
)

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	root := newRootCommand()
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	if err := root.ExecuteContext(ctx); err != nil {
		return exitFailure
	}
	return exitSuccess
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "chancery",
		Short: "Configuration-driven AI gateway",
		Args:  cobra.NoArgs,
		RunE:  runRootCommand,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().String("config", "", "path to the external config directory")
	if err := root.MarkPersistentFlagRequired("config"); err != nil {
		panic(fmt.Errorf("mark config flag required: %w", err))
	}
	root.AddCommand(
		newValidateCommand(),
		newListCommand(),
		newCallCommand(),
		newServeCommand(),
	)
	return root
}

func runRootCommand(_ *cobra.Command, _ []string) error {
	return errors.New("a command is required")
}

func commandConfigPath(command *cobra.Command) (string, error) {
	configPath, err := command.Root().PersistentFlags().GetString("config")
	if err != nil {
		return "", fmt.Errorf("read config flag: %w", err)
	}
	return configPath, nil
}
