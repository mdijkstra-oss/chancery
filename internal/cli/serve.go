package cli

import (
	"github.com/mdijkstra-oss/chancery/internal/server"

	"github.com/spf13/cobra"
)

func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP gateway",
		Args:  cobra.NoArgs,
		RunE:  runServeCommand,
	}
}

func runServeCommand(command *cobra.Command, _ []string) error {
	configPath, err := commandConfigPath(command)
	if err != nil {
		return err
	}
	registry, err := loadRegistry(configPath)
	if err != nil {
		return err
	}
	return server.Run(command.Context(), registry)
}
