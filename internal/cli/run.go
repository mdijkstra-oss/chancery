package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/mdijkstra-oss/chancery/internal/prompts"

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
		Short: "Turns a directory of Markdown into HTTP endpoints",
		Args:  cobra.NoArgs,
		RunE:  runRootCommand,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	// Cobra prints the whole flag listing after any error a command returns, which
	// reads as though the flags were the problem when what failed was a config
	// diagnostic or a backend. The error says what went wrong; --help says the rest.
	root.SilenceUsage = true
	root.PersistentFlags().String("config", defaultConfigDir, "path to the config directory")
	root.PersistentFlags().String("models", prompts.DefaultModelsFile, "alias table to read, relative to the config directory")
	root.AddCommand(
		newValidateCommand(),
		newListCommand(),
		newCallCommand(),
		newServeCommand(),
		newHealthcheckCommand(),
	)
	return root
}

func runRootCommand(_ *cobra.Command, _ []string) error {
	return errors.New("a command is required")
}

// defaultConfigDir is where a directory of agents sits when nobody says otherwise.
// A missing one is reported at boot naming the path, so a wrong working directory
// says so rather than serving no routes.
const defaultConfigDir = "./config"

type configLocation struct {
	Root       string
	ModelsFile string
}

func commandConfigLocation(command *cobra.Command) (configLocation, error) {
	flags := command.Root().PersistentFlags()
	root, err := flags.GetString("config")
	if err != nil {
		return configLocation{}, fmt.Errorf("read config flag: %w", err)
	}
	modelsFile, err := flags.GetString("models")
	if err != nil {
		return configLocation{}, fmt.Errorf("read models flag: %w", err)
	}
	return configLocation{Root: root, ModelsFile: modelsFile}, nil
}
