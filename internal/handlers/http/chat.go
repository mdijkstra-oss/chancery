package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"hermes-logos/internal/prompts"
)

func NewChatHandler(cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, cfg)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request, cfg Config) {
	subfolder := chi.URLParam(r, "subfolder")
	promptPath := filepath.Join(cfg.PromptsBaseDir, subfolder)

	systemPrompt, err := prompts.Load(promptPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	openaiReq := buildOpenAIRequest(cfg.Model, systemPrompt, cfg.GPTVerbosity, cfg.ReasoningEffort, cfg.Tools, req.Messages)

	logOutgoingRequest(openaiReq, cfg.Verbose)

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

	streamResponse(w, resp, cfg)
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

func streamResponse(w http.ResponseWriter, resp *http.Response, cfg Config) {
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	streamWithUsageLogging(resp.Body, w, flusher, cfg.Pricing, cfg.Verbose)
}
