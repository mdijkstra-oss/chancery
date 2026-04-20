package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"hermes-logos/internal/pipeline"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/providers"
	"hermes-logos/internal/providers/sse"
	"hermes-logos/internal/ratelimit"
	"hermes-logos/internal/telemetry"
)

func NewChatHandler(registry prompts.Registry, limiter *ratelimit.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, registry, limiter)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request, registry prompts.Registry, limiter *ratelimit.Limiter) {
	urlPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")

	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	params, promptCfg, err := pipeline.BuildRequestParams(urlPath, req, registry)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if tc := r.URL.Query().Get("tool_choice"); tc != "" {
		params.ToolChoice = tc
	}
	if t := parseTemperature(r.URL.Query().Get("temperature")); t != nil {
		params.Temperature = t
	}
	if rs := r.URL.Query().Get("reasoning_summary"); rs != "" {
		params.ReasoningSummary = rs
	}

	trigger := telemetry.DeriveTrigger(req.Messages)

	slog.InfoContext(r.Context(), "request expanded",
		"component", "chat",
		slog.Group("data",
			slog.String("endpoint", urlPath),
			slog.String("model", params.Model),
			slog.Int("messages", len(req.Messages)),
			slog.Int("tools", len(req.Tools)),
		),
	)

	start := time.Now()
	streamFn := providers.StreamForProtocol(promptCfg.Provider.Protocol)
	result, err := ratelimit.Do(r.Context(), limiter, promptCfg.Provider.Key, 3, func() (sse.StreamResult, error) {
		return streamFn(r.Context(), w, params, promptCfg.Provider)
	})
	duration := time.Since(start).Milliseconds()
	if err != nil {
		slog.ErrorContext(r.Context(), "stream error",
			"component", "chat",
			"error", err,
			slog.Group("data",
				slog.String("endpoint", urlPath),
				slog.String("model", params.Model),
			),
		)
		return
	}

	rec := telemetry.BuildCallRecord(urlPath, params.Model, promptCfg.ReasoningEffort, promptCfg.ServiceTier, trigger, result.Usage, promptCfg.Pricing, duration)
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
