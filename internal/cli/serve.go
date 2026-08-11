package cli

import (
	"github.com/mdijkstra-oss/chancery/internal/server"

	"github.com/spf13/cobra"
)

func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Serve the configured agents over HTTP",
		Args:  cobra.NoArgs,
		RunE:  runServeCommand,
	}
}

func runServeCommand(command *cobra.Command, _ []string) error {
	location, err := commandConfigLocation(command)
	if err != nil {
		return err
	}
	registry, err := loadRegistry(location)
	if err != nil {
		return err
	}
	return server.Run(command.Context(), registry)
}
