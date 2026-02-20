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

	toolNames := ExtractToolNames(req.Tools)

	toolPrompt, toolSources, err := prompts.LoadToolPrompts(filepath.Join(prompts.PromptsDir, "tools"), toolNames)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fullPrompt := agent.Prompt
	if toolPrompt != "" {
		fullPrompt = fullPrompt + "\n\n" + toolPrompt
	}

	allSources := append(agent.Sources, toolSources...)

	toolChoice := r.URL.Query().Get("tool_choice")
	temperature := parseTemperature(r.URL.Query().Get("temperature"))
	apiReq := buildResponsesRequest(promptCfg.Model, fullPrompt, promptCfg.ReasoningEffort, promptCfg.ReasoningSummary, promptCfg.Verbosity, req.Tools, toolChoice, temperature, req.Messages, req.ResponseFormat)

	logOutgoingRequest(apiReq, cfg.Verbose)

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

	streamResponse(w, resp, cfg, urlPath, toolNames, allSources, promptCfg.Pricing, promptCfg.ReasoningEffort)
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

func streamResponse(w http.ResponseWriter, resp *http.Response, cfg Config, endpoint string, toolNames []string, sources []string, pricing prompts.Pricing, reasoningEffort string) {
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	streamWithUsageLogging(resp.Body, w, flusher, cfg.Verbose, endpoint, toolNames, sources, pricing, reasoningEffort)
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
