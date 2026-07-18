package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"hermes-logos/internal/auth"
	"hermes-logos/internal/bootstrap"
	"hermes-logos/internal/config"
	httpHandlers "hermes-logos/internal/handlers/http"
	"hermes-logos/internal/pipeline"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/providers"
	"hermes-logos/internal/providers/openai"
	"hermes-logos/internal/providers/sse"
	"hermes-logos/internal/ratelimit"

	"github.com/go-chi/chi/v5"
)

type listModel struct {
	Name            string `json:"name,omitempty"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	Default         bool   `json:"default,omitempty"`
}

type listAgent struct {
	Path        string      `json:"path"`
	Description string      `json:"description,omitempty"`
	Model       string      `json:"model,omitempty"`
	Reasoning   string      `json:"reasoning_effort,omitempty"`
	Models      []listModel `json:"models,omitempty"`
}

type listSummary struct {
	Agents    int `json:"agents"`
	Models    int `json:"models"`
	Providers int `json:"providers"`
}

type listOutput struct {
	Agents  []listAgent `json:"agents"`
	Summary listSummary `json:"summary"`
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	rootFlags := flag.NewFlagSet("hermes-logos", flag.ContinueOnError)
	rootFlags.SetOutput(stderr)
	configPath := rootFlags.String("config", "", "path to the external config directory")
	rootFlags.Usage = func() {
		fmt.Fprintln(stderr, "usage: hermes-logos --config PATH <serve|validate|list|call>")
	}
	if err := rootFlags.Parse(args); err != nil {
		return 2
	}
	if *configPath == "" {
		fmt.Fprintln(stderr, "error: --config PATH is required")
		return 2
	}
	commandArgs := rootFlags.Args()
	if len(commandArgs) == 0 {
		rootFlags.Usage()
		return 2
	}

	registry, report := prompts.Load(*configPath)
	switch commandArgs[0] {
	case "validate":
		if len(commandArgs) != 1 {
			fmt.Fprintln(stderr, "usage: hermes-logos --config PATH validate")
			return 2
		}
		return runValidate(report, stdout)
	case "list":
		return runList(commandArgs[1:], registry, report, stdout, stderr)
	case "call":
		return runCall(ctx, commandArgs[1:], registry, report, stdin, stdout, stderr)
	case "serve":
		return runServe(ctx, commandArgs[1:], registry, report, stderr)
	default:
		fmt.Fprintf(stderr, "error: unknown command %q\n", commandArgs[0])
		rootFlags.Usage()
		return 2
	}
}

func runValidate(report prompts.Report, output io.Writer) int {
	for _, diagnostic := range report.Diagnostics {
		symbol := "!"
		if diagnostic.Severity == prompts.SeverityError {
			symbol = "✗"
		}
		fmt.Fprintf(output, "%s %s: %s\n", symbol, diagnostic.Path, diagnostic.Message)
	}
	if report.HasErrors() {
		fmt.Fprintf(output, "✗ config invalid (%d errors · %d warnings)\n", report.ErrorCount(), report.WarningCount())
		return 1
	}
	fmt.Fprintf(output, "✓ config valid (%d warnings)\n", report.WarningCount())
	return 0
}

func runList(args []string, registry prompts.Registry, report prompts.Report, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	asJSON := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: hermes-logos --config PATH list [--json]")
		return 2
	}
	if report.HasErrors() {
		printLoadErrors(report, stderr)
		return 1
	}
	output := buildListOutput(registry)
	if *asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			fmt.Fprintf(stderr, "encode list: %v\n", err)
			return 1
		}
		return 0
	}
	printList(output, stdout)
	return 0
}

func runCall(ctx context.Context, args []string, registry prompts.Registry, report prompts.Report, stdin io.Reader, stdout, stderr io.Writer) int {
	if report.HasErrors() {
		printLoadErrors(report, stderr)
		return 1
	}
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: hermes-logos --config PATH call <agent-path> [--input TEXT|@FILE]")
		return 2
	}
	reference := args[0]
	flags := flag.NewFlagSet("call", flag.ContinueOnError)
	flags.SetOutput(stderr)
	inputFlag := flags.String("input", "", "input text, @file, or stdin when omitted")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: hermes-logos --config PATH call <agent-path> [--input TEXT|@FILE]")
		return 2
	}
	input, err := readInput(*inputFlag, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	resolved, err := registry.ResolveAgent(reference)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cfg, err := prompts.ResolveAPIKey(resolved.Config, os.Getenv)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if resolved.Path == "embeddings" {
		return callEmbeddings(ctx, input, cfg, stdout, stderr)
	}
	return callChat(ctx, reference, input, cfg, registry, stdout, stderr)
}

func runServe(ctx context.Context, args []string, registry prompts.Registry, report prompts.Report, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: hermes-logos --config PATH serve")
		return 2
	}
	if report.HasErrors() {
		printLoadErrors(report, stderr)
		return 1
	}
	resolvedRegistry, err := registry.WithAPIKeys(os.Getenv)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	embeddings, err := resolvedRegistry.ResolveAgent("embeddings")
	if err != nil {
		fmt.Fprintln(stderr, "embeddings agent is required for serve")
		return 1
	}

	runtimeConfig, err := config.Load()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	bootstrap.SetupLogger(runtimeConfig.LogLevel, runtimeConfig.Environment)
	validator, err := auth.NewValidator(ctx, runtimeConfig.Auth)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer validator.Close()
	if !validator.Enabled() {
		slog.Warn("auth disabled — all requests accepted")
	}
	slog.Info("config loaded", "component", "startup", slog.Group("data", slog.Int("agents", len(resolvedRegistry.Agents))))
	limiter := ratelimit.NewLimiter()
	chatHandler := httpHandlers.NewChatHandler(resolvedRegistry, limiter)
	embeddingsHandler := httpHandlers.NewEmbeddingsHandler(embeddings.Config, limiter)
	router := chi.NewRouter()
	httpHandlers.SetupRoutes(router, chatHandler, embeddingsHandler, httpHandlers.JWTAuthentication(validator), runtimeConfig.CorsOrigins, runtimeConfig.RequestHeaders)
	address := ":" + runtimeConfig.Port
	slog.Info("server starting", "component", "startup", slog.Group("data", slog.String("port", runtimeConfig.Port)))
	if err := http.ListenAndServe(address, router); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func callChat(ctx context.Context, reference, input string, cfg prompts.PromptConfig, registry prompts.Registry, stdout, stderr io.Writer) int {
	message, err := json.Marshal(protocol.InputMessage{Type: "message", Role: "user", Content: input})
	if err != nil {
		fmt.Fprintf(stderr, "build input: %v\n", err)
		return 1
	}
	request := protocol.ChatRequest{Messages: []json.RawMessage{message}}
	params, _, err := pipeline.BuildRequestParams(reference, request, registry)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	writer := sse.NewTextWriter(stdout)
	stream := providers.StreamForProtocol(cfg.Provider.Protocol)
	if _, err := stream(ctx, writer, params, cfg.Provider); err != nil {
		fmt.Fprintf(stderr, "call failed: %v\n", err)
		return 1
	}
	if err := writer.Close(); err != nil {
		fmt.Fprintf(stderr, "write output: %v\n", err)
		return 1
	}
	return 0
}

func callEmbeddings(ctx context.Context, input string, cfg prompts.PromptConfig, stdout, stderr io.Writer) int {
	result, err := openai.Embed(ctx, []string{input}, cfg.Model, cfg.Dimensions, cfg.Provider)
	if err != nil {
		fmt.Fprintf(stderr, "call failed: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(result.Body); err != nil {
		fmt.Fprintf(stderr, "write output: %v\n", err)
		return 1
	}
	return 0
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

func buildListOutput(registry prompts.Registry) listOutput {
	agents := make([]listAgent, 0, len(registry.Agents))
	for _, path := range registry.AgentPaths() {
		agent := listAgent{Path: path, Description: registry.Descriptions[path]}
		if named := registry.NamedConfigs[path]; len(named) > 0 {
			names := make([]string, 0, len(named))
			for name := range named {
				names = append(names, name)
			}
			slices.Sort(names)
			for _, name := range names {
				cfg := named[name]
				agent.Models = append(agent.Models, listModel{Name: name, Model: cfg.Model, ReasoningEffort: cfg.ReasoningEffort, Default: registry.Defaults[path] == name})
			}
		} else {
			cfg := registry.Configs[path]
			agent.Model = cfg.Model
			agent.Reasoning = cfg.ReasoningEffort
		}
		agents = append(agents, agent)
	}
	return listOutput{
		Agents: agents,
		Summary: listSummary{
			Agents:    len(registry.Agents),
			Models:    registry.ModelCount(),
			Providers: registry.ProviderCount(),
		},
	}
}

func printList(output listOutput, destination io.Writer) {
	writer := tabwriter.NewWriter(destination, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "PATH\tMODEL\tREASONING")
	for _, agent := range output.Agents {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", agent.Path, agent.Model, agent.Reasoning)
		for _, model := range agent.Models {
			marker := ""
			if model.Default {
				marker = " (default)"
			}
			fmt.Fprintf(writer, "  .%s%s\t%s\t%s\n", model.Name, marker, model.Model, model.ReasoningEffort)
		}
	}
	writer.Flush()
	fmt.Fprintf(destination, "%d agents · %d models · %d providers\n", output.Summary.Agents, output.Summary.Models, output.Summary.Providers)
}

func printLoadErrors(report prompts.Report, destination io.Writer) {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Severity == prompts.SeverityError {
			fmt.Fprintf(destination, "✗ %s: %s\n", diagnostic.Path, diagnostic.Message)
		}
	}
}
