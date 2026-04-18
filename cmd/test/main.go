package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"hermes-logos/internal/pipeline"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/providers"
	"hermes-logos/internal/providers/sse"
)

func main() {
	temperature := flag.String("temperature", "", "override temperature")
	model := flag.String("model", "", "override model")
	toolChoice := flag.String("tool-choice", "", "override tool choice")
	reasoningSummary := flag.String("reasoning-summary", "", "override reasoning summary")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: test <agent> <messages.json>\n")
		os.Exit(1)
	}
	agentName := args[0]
	messagesFile := args[1]

	registry := prompts.CompileRegistry(prompts.PromptsDir)

	req, err := loadRequest(messagesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load request: %v\n", err)
		os.Exit(1)
	}

	params, promptCfg, err := pipeline.BuildRequestParams(agentName, req, registry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build params: %v\n", err)
		os.Exit(1)
	}

	applyOverrides(&params, *temperature, *model, *toolChoice, *reasoningSummary)

	collector := &sse.Collector{}
	streamFn := providers.StreamForProtocol(promptCfg.Provider.Protocol)
	result, err := streamFn(context.Background(), collector, params, promptCfg.Provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "stream error: %v\n", err)
		os.Exit(1)
	}

	response := collector.Result()
	fmt.Print(response.Text)

	if len(response.ToolCalls) > 0 {
		fmt.Fprintln(os.Stderr, "\n--- tool calls ---")
		for _, tc := range response.ToolCalls {
			fmt.Fprintf(os.Stderr, "%s(%s): %s\n", tc.Name, tc.CallID, tc.Arguments)
		}
	}

	if result.Usage != nil {
		fmt.Fprintf(os.Stderr, "\n--- usage ---")
		fmt.Fprintf(os.Stderr, "\ninput: %d, output: %d, total: %d\n",
			result.Usage.InputTokens, result.Usage.OutputTokens, result.Usage.TotalTokens)
	}
}

func loadRequest(path string) (protocol.ChatRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return protocol.ChatRequest{}, err
	}
	var req protocol.ChatRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return protocol.ChatRequest{}, err
	}
	return req, nil
}

func applyOverrides(params *protocol.RequestParams, temperature, model, toolChoice, reasoningSummary string) {
	if temperature != "" {
		v, err := strconv.ParseFloat(temperature, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid temperature: %v\n", err)
			os.Exit(1)
		}
		params.Temperature = &v
	}
	if model != "" {
		params.Model = model
	}
	if toolChoice != "" {
		params.ToolChoice = toolChoice
	}
	if reasoningSummary != "" {
		params.ReasoningSummary = reasoningSummary
	}
}
