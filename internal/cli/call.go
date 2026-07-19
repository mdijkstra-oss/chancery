package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/matthijn/hermes-logos/internal/invoke"
	"github.com/matthijn/hermes-logos/internal/prompts"
	"github.com/matthijn/hermes-logos/internal/protocol"
	"github.com/matthijn/hermes-logos/internal/providers/sse"

	"github.com/spf13/cobra"
)

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
	configPath, err := commandConfigPath(command)
	if err != nil {
		return err
	}
	inputValue, err := command.Flags().GetString("input")
	if err != nil {
		return fmt.Errorf("read input flag: %w", err)
	}
	registry, err := loadRegistry(configPath)
	if err != nil {
		return err
	}
	input, err := readInput(inputValue, command.InOrStdin())
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	target, err := invoke.Resolve(args[0], registry, os.Getenv)
	if err != nil {
		return err
	}
	switch target.Kind {
	case invoke.KindChat:
		return writeChat(command, input, target, registry)
	case invoke.KindEmbeddings:
		return writeEmbeddings(command, input, target)
	default:
		panic("unknown invocation kind: " + target.Kind)
	}
}

func writeChat(command *cobra.Command, input string, target invoke.Target, registry prompts.Registry) error {
	message, err := json.Marshal(protocol.InputMessage{Type: "message", Role: "user", Content: input})
	if err != nil {
		return fmt.Errorf("build input: %w", err)
	}
	request := protocol.ChatRequest{Messages: []json.RawMessage{message}}
	writer := sse.NewTextWriter(command.OutOrStdout())
	if _, err := invoke.Chat(command.Context(), target, request, registry, writer); err != nil {
		return fmt.Errorf("call failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func writeEmbeddings(command *cobra.Command, input string, target invoke.Target) error {
	result, err := invoke.Embeddings(command.Context(), target, []string{input})
	if err != nil {
		return fmt.Errorf("call failed: %w", err)
	}
	if _, err := command.OutOrStdout().Write(result.Body); err != nil {
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
