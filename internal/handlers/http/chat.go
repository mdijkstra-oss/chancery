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
)

func NewChatHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, cfg)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request, cfg Config) {
	urlPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	chat := r.URL.Query().Get("chat") == "true"

	resolved, err := config.ResolveFolder(urlPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	promptCfg, err := prompts.ResolveConfig(config.PromptsDir, resolved.Folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	toolNames := ExtractToolNames(req.Tools)

	composed, err := prompts.ComposePrompt(config.PromptsDir, prompts.ComposeOpts{
		Folder: resolved.Folder,
		Tools:  toolNames,
		Chat:   chat,
		Extra:  resolved.Extra,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	toolChoice := r.URL.Query().Get("tool_choice")
	temperature := parseTemperature(r.URL.Query().Get("temperature"))
	apiReq := buildResponsesRequest(promptCfg.Model, composed.Prompt, promptCfg.ReasoningEffort, promptCfg.ReasoningSummary, promptCfg.Verbosity, req.Tools, toolChoice, temperature, req.Messages, req.ResponseFormat)

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

	streamResponse(w, resp, cfg, urlPath, toolNames, composed.Sources, promptCfg.Pricing)
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

func streamResponse(w http.ResponseWriter, resp *http.Response, cfg Config, endpoint string, toolNames []string, sources []string, pricing prompts.Pricing) {
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	streamWithUsageLogging(resp.Body, w, flusher, cfg.Verbose, endpoint, toolNames, sources, pricing)
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
