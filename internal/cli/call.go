package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mdijkstra-oss/chancery/internal/config"
	"github.com/mdijkstra-oss/chancery/internal/responses"

	"github.com/spf13/cobra"
)

// inputItem is one item of the Responses input array. A terminal gives one turn of
// text, which is the smallest body the format accepts.
type inputItem struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

type callRequest struct {
	Input  []inputItem `json:"input"`
	Stream bool        `json:"stream"`
}

func newCallCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "call <agent-path>",
		Short: "Call a configured agent",
		Args:  cobra.ExactArgs(1),
		RunE:  runCallCommand,
	}
	command.Flags().String("input", "", "input text, @file, or stdin when omitted")
	return command
}

func runCallCommand(command *cobra.Command, args []string) error {
	location, err := commandConfigLocation(command)
	if err != nil {
		return err
	}
	inputValue, err := command.Flags().GetString("input")
	if err != nil {
		return fmt.Errorf("read input flag: %w", err)
	}
	registry, err := loadRegistry(location)
	if err != nil {
		return err
	}
	input, err := readInput(inputValue, command.InOrStdin())
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	resolved, err := registry.ResolveAgent(args[0])
	if err != nil {
		return err
	}
	backend, err := config.LoadBackend()
	if err != nil {
		return err
	}
	client, err := responses.NewClient(backend)
	if err != nil {
		return err
	}

	body, err := composeCall(input, responses.AgentFrom(resolved))
	if err != nil {
		return err
	}
	response, err := client.Send(command.Context(), responses.Request{Body: body})
	if err != nil {
		return fmt.Errorf("call failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return responses.StatusErrorFrom(response)
	}
	// A backend that answered and then went quiet would otherwise hold the terminal
	// open for as long as it stays connected.
	stream := responses.WithStallTimeout(response.Body, responses.StallTimeout)
	defer stream.Close()
	return renderStream(command.OutOrStdout(), stream)
}

func composeCall(input string, agent responses.Agent) ([]byte, error) {
	request := callRequest{
		Input:  []inputItem{{Type: "message", Role: "user", Content: input}},
		Stream: true,
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("build input: %w", err)
	}
	composed, err := responses.Compose(encoded, agent)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	return composed, nil
}

// A stream that ends in response.failed, or ends without response.completed at all,
// leaves Close holding the error that becomes the exit code.
func renderStream(destination io.Writer, body io.Reader) error {
	writer := newTextWriter(destination)
	if _, err := io.Copy(writer, body); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func readInput(value string, stdin io.Reader) (string, error) {
	if strings.HasPrefix(value, "@") {
		data, err := os.ReadFile(strings.TrimPrefix(value, "@"))
		return string(data), err
	}
	if value != "" {
		return value, nil
	}
	data, err := io.ReadAll(stdin)
	return string(data), err
}
