package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"hermes-logos/internal/prompts"
)

func NewChatHandler(cfg Config, registry prompts.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, cfg, registry)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request, cfg Config, registry prompts.Registry) {
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

	req.Messages = ExpandMessages(req.Messages, registry.Modes)

	var reasoningEffort string
	req.Messages, reasoningEffort = ExtractReasoningEffort(req.Messages)
	if reasoningEffort == "" {
		reasoningEffort = promptCfg.ReasoningEffort
	}

	var model string
	req.Messages, model = ExtractModel(req.Messages)
	if model == "" {
		model = promptCfg.Model
	}

	var verbosity string
	req.Messages, verbosity = ExtractVerbosity(req.Messages)
	if verbosity == "" {
		verbosity = promptCfg.Verbosity
	}

	toolNames := ExtractToolNames(req.Tools)

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
	reasoningSummary := r.URL.Query().Get("reasoning_summary")
	if reasoningSummary == "" {
		reasoningSummary = promptCfg.ReasoningSummary
	}
	apiReq := buildResponsesRequest(model, fullPrompt, reasoningEffort, reasoningSummary, verbosity, promptCfg.ServiceTier, req.Tools, toolChoice, temperature, req.Messages, req.ResponseFormat)

	forceCompact := r.URL.Query().Get("compact") == "true"
	tokens := estimateTokens(apiReq.Input)
	if forceCompact || shouldCompact(promptCfg.CompactAt, tokens) {
		compacterAgent, compacterOk := registry.Agents["compacter"]
		compacterCfg, compacterCfgOk := registry.Configs["compacter"]
		if !compacterOk || !compacterCfgOk {
			slog.Error("compacter agent not found in registry")
		} else {
			compactReq := buildCompactRequest(compacterCfg.Model, compacterAgent.Prompt, stripForCompaction(req.Messages))
			resp, err := proxyWithRetry(r.Context(), compactReq, cfg)
			if err != nil {
				handleProxyError(w, err)
				return
			}
			defer resp.Body.Close()
			if isErrorResponse(resp) {
				handleUpstreamError(w, resp)
				return
			}
			slog.Info("compacting", "tokens", tokens, "compact_at", promptCfg.CompactAt)
			if cfg.Inspect {
				inspectJSON("compacter request", compactReq)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			flusher := w.(http.Flusher)
			usage, err := streamCompaction(resp.Body, w, flusher)
			if err != nil {
				slog.Error("compaction stream error", "error", err)
				return
			}
			if usage != nil {
				logUsage("compacter", usage, compacterCfg.Pricing, "", 0)
			}
			return
		}
	}

	if cfg.Inspect {
		inspectJSON(urlPath+" request", apiReq)
	}

	resp, err := proxyWithRetry(r.Context(), apiReq, cfg)
	if err != nil {
		handleProxyError(w, err)
		return
	}
	defer resp.Body.Close()

	if isErrorResponse(resp) {
		handleUpstreamError(w, resp)
		return
	}

	streamResponse(w, resp, cfg, urlPath, promptCfg.Pricing, reasoningEffort, tokens)
}

func decodeRequest(r *http.Request) (ChatRequest, error) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	if len(req.Messages) == 0 {
		return req, http.ErrBodyNotAllowed
	}
	return req, nil
}

var (
	errQuotaExhausted = errors.New("API quota exhausted")
	errRateLimited    = errors.New("rate limit exceeded after retries")
)

const maxUpstreamRetries = 3

type apiErrorBody struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func isQuotaError(body []byte) bool {
	var parsed apiErrorBody
	if json.Unmarshal(body, &parsed) != nil {
		return false
	}
	switch parsed.Error.Code {
	case "insufficient_quota", "billing_hard_limit_reached":
		return true
	}
	return false
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<uint(attempt)) * time.Second
}

func proxyRequest(ctx context.Context, apiReq ResponsesRequest, cfg Config) (*http.Response, error) {
	proxyReq, err := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+"/responses", jsonReader(apiReq))
	if err != nil {
		return nil, err
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	return http.DefaultClient.Do(proxyReq)
}

func proxyWithRetry(ctx context.Context, apiReq ResponsesRequest, cfg Config) (*http.Response, error) {
	for attempt := range maxUpstreamRetries {
		resp, err := proxyRequest(ctx, apiReq, cfg)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if isQuotaError(body) {
			return nil, errQuotaExhausted
		}

		if attempt == maxUpstreamRetries-1 {
			return nil, errRateLimited
		}

		delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
		slog.Warn("rate limited, retrying", "attempt", attempt+1, "delay", delay)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, errRateLimited
}

func handleProxyError(w http.ResponseWriter, err error) {
	if errors.Is(err, errQuotaExhausted) {
		slog.Error("quota exhausted")
		http.Error(w, "API quota exhausted", http.StatusPaymentRequired)
		return
	}
	if errors.Is(err, errRateLimited) {
		slog.Error("rate limited after retries")
		http.Error(w, "Rate limited after retries", http.StatusTooManyRequests)
		return
	}
	slog.Error("upstream request failed", "error", err)
	http.Error(w, "upstream request failed", http.StatusBadGateway)
}

func isErrorResponse(resp *http.Response) bool {
	return resp.StatusCode >= 400
}

func handleUpstreamError(w http.ResponseWriter, resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	slog.Error("upstream error", "status", resp.StatusCode, "body", string(body))
	http.Error(w, string(body), resp.StatusCode)
}

func streamResponse(w http.ResponseWriter, resp *http.Response, cfg Config, endpoint string, pricing prompts.Pricing, reasoningEffort string, estimatedTokens int) {
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	streamWithUsageLogging(resp.Body, w, flusher, cfg, endpoint, pricing, reasoningEffort, estimatedTokens)
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
