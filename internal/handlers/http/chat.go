package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

func NewChatHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, cfg)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request, cfg Config) {
	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	enableCaching := shouldEnablePromptCaching(cfg.Model)
	messagesWithCache, breakpoints := addCacheBreakpoints(req.Messages, enableCaching, cfg.CacheInterval)

	openaiReq := buildOpenAIRequest(cfg.Model, cfg.Provider, cfg.SystemPrompt, cfg.Tools, messagesWithCache, cfg.IncludeReasoning, enableCaching)

	breakdown := calculateTokenBreakdown(openaiReq, cfg.SystemPrompt)
	totalEstimated := sumTokens(breakdown)

	if exceedsTokenWindow(totalEstimated, cfg.MaxTokenWindow) {
		http.Error(w, "Out of context, please reload", http.StatusInternalServerError)
		return
	}

	logOutgoingRequest(openaiReq, breakdown, cfg.Verbose)

	resp, err := proxyRequest(r.Context(), openaiReq, cfg)
	if err != nil {
		handleProxyError(w, err)
		return
	}
	defer resp.Body.Close()

	if isErrorResponse(resp) {
		handleUpstreamError(w, resp)
		return
	}

	streamResponse(w, resp, breakdown, breakpoints, enableCaching, len(openaiReq.Tools) > 0, cfg.Verbose)
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

func exceedsTokenWindow(total, max int) bool {
	return max > 0 && total > max
}

func proxyRequest(ctx context.Context, openaiReq OpenAIRequest, cfg Config) (*http.Response, error) {
	proxyReq, err := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+"/chat/completions", jsonReader(openaiReq))
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

func streamResponse(w http.ResponseWriter, resp *http.Response, breakdown TokenBreakdown, breakpoints []CacheBreakpointInfo, enableCaching, hasTools, verbose bool) {
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	streamWithUsageLogging(resp.Body, w, flusher, breakdown, breakpoints, enableCaching, hasTools, verbose)
}
