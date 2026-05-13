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
	"hermes-logos/internal/telemetry"
)

func main() {
	temperature := flag.String("temperature", "", "override temperature")
	model := flag.String("model", "", "override model")
	toolChoice := flag.String("tool-choice", "", "override tool choice")
	reasoning := flag.String("reasoning", "", "override reasoning effort (low, medium, high)")
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

	params, promptCfg, err := pipeline.BuildRequestParams(agentName, 0, req, registry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build params: %v\n", err)
		os.Exit(1)
	}

	applyOverrides(&params, *temperature, *model, *toolChoice, *reasoning, *reasoningSummary)

	if *model != "" {
		override, ok := registry.LookupModel(*model)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown model: %s\n", *model)
			os.Exit(1)
		}
		params.Model = override.Name
		promptCfg.Provider = override.Provider
		promptCfg.Pricing = override.Pricing
	}

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

	fmt.Fprintf(os.Stderr, "\n--- meta ---\n")
	fmt.Fprintf(os.Stderr, "model: %s\n", params.Model)
	if params.ReasoningEffort != "" {
		fmt.Fprintf(os.Stderr, "reasoning: %s\n", params.ReasoningEffort)
	}
	if result.Usage != nil {
		rec := telemetry.BuildCallRecord("chat", params.Model, "", "", "", result.Usage, promptCfg.Pricing, 0)
		fmt.Fprintf(os.Stderr, "input: %d, output: %d, total: %d\n",
			result.Usage.InputTokens, result.Usage.OutputTokens, result.Usage.TotalTokens)
		fmt.Fprintf(os.Stderr, "cost: $%.4f (input: $%.4f, cached: $%.4f, output: $%.4f, reasoning: $%.4f)\n",
			rec.TotalCost, rec.InputCost, rec.CachedInputCost, rec.OutputCost, rec.ReasoningCost)
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

func applyOverrides(params *protocol.RequestParams, temperature, model, toolChoice, reasoning, reasoningSummary string) {
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
	if reasoning != "" {
		params.ReasoningEffort = reasoning
	}
	if reasoningSummary != "" {
		params.ReasoningSummary = reasoningSummary
	}
}
