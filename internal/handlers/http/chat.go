package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mdijkstra-oss/chancery/internal/auth"
	"github.com/mdijkstra-oss/chancery/internal/pipeline"
	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/protocol"
	"github.com/mdijkstra-oss/chancery/internal/providers"
	"github.com/mdijkstra-oss/chancery/internal/providers/sse"
	"github.com/mdijkstra-oss/chancery/internal/quota"
	"github.com/mdijkstra-oss/chancery/internal/ratelimit"
	"github.com/mdijkstra-oss/chancery/internal/telemetry"

	"github.com/go-chi/chi/v5"
)

func NewChatHandler(registry prompts.Registry, limiter *ratelimit.Limiter, quotaClient *quota.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, registry, limiter, quotaClient)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request, registry prompts.Registry, limiter *ratelimit.Limiter, quotaClient *quota.Client) {
	urlPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
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

	quotaRequest := buildChatQuotaRequest(RequestIDFromContext(r.Context()), auth.UserFromContext(r.Context()), urlPath, params, promptCfg)
	reservation, allowed := reserveQuota(r.Context(), w, quotaClient, quotaRequest)
	if !allowed {
		return
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
	result, err := ratelimit.Do(r.Context(), limiter, params.Model, 3, func() (sse.StreamResult, error) {
		return streamFn(r.Context(), w, params, promptCfg.Provider)
	})
	duration := time.Since(start).Milliseconds()
	if err != nil {
		settleQuota(r.Context(), quotaClient, reservation, failedQuotaOutcome(r.Context()), result.Usage)
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

	settleQuota(r.Context(), quotaClient, reservation, quota.OutcomeCompleted, result.Usage)
	rec := telemetry.BuildCallRecord(urlPath, params.Model, promptCfg.ReasoningEffort, promptCfg.ServiceTier, trigger, result.Usage, duration)
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
