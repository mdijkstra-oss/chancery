package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"hermes-logos/internal/config"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/tools"
)

func NewChatHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, cfg)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request, cfg Config) {
	endpoint := strings.TrimPrefix(chi.URLParam(r, "*"), "/")

	endpointCfg, ok := config.GetEndpoint(endpoint)
	if !ok {
		http.Error(w, "unknown endpoint: "+endpoint, http.StatusNotFound)
		return
	}

	promptCfg, err := prompts.LoadConfig(config.PromptsDir, endpointCfg.Folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	systemPrompt, err := prompts.LoadFolder(config.PromptsDir, endpointCfg.Folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var loadedTools []json.RawMessage
	if endpointCfg.IncludeTools {
		loadedTools, _ = tools.LoadFolder(config.PromptsDir, endpointCfg.Folder)
	}

	toolChoice := r.URL.Query().Get("tool_choice")
	temperature := parseTemperature(r.URL.Query().Get("temperature"))
	apiReq := buildResponsesRequest(promptCfg.Model, systemPrompt, promptCfg.ReasoningEffort, promptCfg.Verbosity, loadedTools, toolChoice, temperature, req.Messages)

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

	streamResponse(w, resp, cfg, promptCfg.Pricing)
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

func streamResponse(w http.ResponseWriter, resp *http.Response, cfg Config, pricing prompts.Pricing) {
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	streamWithUsageLogging(resp.Body, w, flusher, cfg.Verbose, pricing)
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
