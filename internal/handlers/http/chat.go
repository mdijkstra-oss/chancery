package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"hermes-logos/internal/messages"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/providers"
	"hermes-logos/internal/telemetry"
)

func NewChatHandler(inspect bool, registry prompts.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, inspect, registry)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request, inspect bool, registry prompts.Registry) {
	urlPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")

	agent, ok := registry.Agents[urlPath]
	if !ok {
		http.Error(w, "unknown agent: "+urlPath, http.StatusNotFound)
		return
	}

	promptCfg, ok := registry.Configs[urlPath]
	if !ok {
		http.Error(w, "no config for agent: "+urlPath, http.StatusInternalServerError)
		return
	}

	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.Messages = messages.ExpandMessages(req.Messages, registry.Modes)
	req.Messages = messages.ExpandApproaches(req.Messages, registry.Approaches.Entries)

	model := promptCfg.Model
	toolNames := protocol.ExtractToolNames(req.Tools)

	toolPrompt, _, err := prompts.LoadToolPrompts(filepath.Join(prompts.PromptsDir, "tools"), toolNames)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fullPrompt := agent.Prompt
	if toolPrompt != "" {
		fullPrompt = fullPrompt + "\n\n" + toolPrompt
	}

	toolChoice := r.URL.Query().Get("tool_choice")
	temperature := parseTemperature(r.URL.Query().Get("temperature"))
	if temperature == nil {
		temperature = promptCfg.Temperature
	}
	reasoningSummary := r.URL.Query().Get("reasoning_summary")
	if reasoningSummary == "" {
		reasoningSummary = promptCfg.ReasoningSummary
	}

	params := protocol.RequestParams{
		Model:            model,
		SystemPrompt:     fullPrompt,
		ReasoningEffort:  promptCfg.ReasoningEffort,
		ReasoningSummary: reasoningSummary,
		Verbosity:        promptCfg.Verbosity,
		ServiceTier:      promptCfg.ServiceTier,
		ToolChoice:       toolChoice,
		LegacyThinking:  promptCfg.LegacyThinking,
		Temperature:      temperature,
		Tools:            req.Tools,
		Messages:         req.Messages,
		ResponseFormat:   req.ResponseFormat,
	}

	if inspect {
		apiReq := protocol.BuildResponsesRequestFromParams(params)
		inspectJSON(urlPath+" request", apiReq)
	}

	trigger := telemetry.DeriveTrigger(req.Messages)

	slog.InfoContext(r.Context(), "request expanded",
		"component", "chat",
		slog.Group("data",
			slog.String("endpoint", urlPath),
			slog.String("model", model),
			slog.Int("messages", len(req.Messages)),
			slog.Int("tools", len(req.Tools)),
		),
	)

	start := time.Now()
	streamFn := providers.StreamForProtocol(promptCfg.Provider.Protocol)
	result, err := streamFn(r.Context(), w, params, promptCfg.Provider)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		slog.ErrorContext(r.Context(), "stream error",
			"component", "chat",
			"error", err,
			slog.Group("data",
				slog.String("endpoint", urlPath),
				slog.String("model", model),
			),
		)
		return
	}

	rec := telemetry.BuildCallRecord(urlPath, model, promptCfg.ReasoningEffort, promptCfg.ServiceTier, trigger, result.Usage, promptCfg.Pricing, duration)
	telemetry.LogCallRecord(r.Context(), rec)
}

func decodeRequest(r *http.Request) (protocol.ChatRequest, error) {
	var req protocol.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	if len(req.Messages) == 0 {
		return req, http.ErrBodyNotAllowed
	}
	return req, nil
}

func parseTemperature(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
