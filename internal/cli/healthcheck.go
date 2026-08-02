package cli

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mdijkstra-oss/chancery/internal/config"

	"github.com/spf13/cobra"
)

// healthcheckTimeout is short because the question is whether a listener answers,
// not whether it answers quickly. A check that waits longer than the interval it
// runs on stops reporting the state it was asked about.
const healthcheckTimeout = 2 * time.Second

// newHealthcheckCommand lets the binary check itself. A container built on scratch
// holds one file and no shell, so a runtime healthcheck naming any other command
// names one the image cannot execute — and a deployment that cannot gate on
// readiness sends its first requests to a process that has not finished binding.
//
// It asks only whether this process is serving. Reaching a model is deliberately
// not part of the answer: a container that reports itself unhealthy when a provider
// is having a bad afternoon gets restarted for someone else's outage.
func newHealthcheckCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "healthcheck",
		Short: "Report whether this process is serving",
		Args:  cobra.NoArgs,
		RunE:  runHealthcheckCommand,
	}
	command.Flags().String("addr", "127.0.0.1:"+config.ListenPort(), "address to check")
	return command
}

func runHealthcheckCommand(command *cobra.Command, _ []string) error {
	addr, err := command.Flags().GetString("addr")
	if err != nil {
		return fmt.Errorf("read addr flag: %w", err)
	}
	request, err := http.NewRequestWithContext(
		command.Context(), http.MethodGet, "http://"+addr+"/health", nil)
	if err != nil {
		return fmt.Errorf("build health request: %w", err)
	}
	client := &http.Client{Timeout: healthcheckTimeout}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("reach %s: %w", addr, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", addr, response.Status)
	}
	return nil
}
