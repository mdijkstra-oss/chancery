package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

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
	apiReq := buildResponsesRequest(promptCfg.Model, fullPrompt, reasoningEffort, reasoningSummary, promptCfg.Verbosity, req.Tools, toolChoice, temperature, req.Messages, req.ResponseFormat)

	forceCompact := r.URL.Query().Get("compact") == "true"
	tokens := estimateTokens(apiReq.Input)
	if forceCompact || shouldCompact(promptCfg.CompactAt, tokens) {
		compacterAgent, compacterOk := registry.Agents["compacter"]
		compacterCfg, compacterCfgOk := registry.Configs["compacter"]
		if !compacterOk || !compacterCfgOk {
			slog.Error("compacter agent not found in registry")
		} else {
			compactReq := buildCompactRequest(compacterCfg.Model, compacterAgent.Prompt, stripForCompaction(req.Messages))
			resp, err := proxyRequest(r.Context(), compactReq, cfg)
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

	resp, err := proxyRequest(r.Context(), apiReq, cfg)
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

func proxyRequest(ctx context.Context, apiReq ResponsesRequest, cfg Config) (*http.Response, error) {
	proxyReq, err := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+"/responses", jsonReader(apiReq))
	if err != nil {
		return nil, err
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	return http.DefaultClient.Do(proxyReq)
}

func handleProxyError(w http.ResponseWriter, err error) {
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
