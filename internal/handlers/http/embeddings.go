package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"hermes-logos/internal/prompts"
)

const maxEmbeddingBatchSize = 512
const maxEmbeddingBatchTokens = 200_000
const charsPerToken = 4

type EmbeddingConfig struct {
	Provider   prompts.ProviderConfig
	Model      string
	Dimensions int
	Pricing    prompts.Pricing
}

type EmbeddingsClientRequest struct {
	Input []string `json:"input"`
}

type embeddingsProxyRequest struct {
	Input      []string `json:"input"`
	Model      string   `json:"model"`
	Dimensions int      `json:"dimensions"`
}

type EmbeddingsResponse struct {
	Data  []EmbeddingData `json:"data"`
	Usage EmbeddingsUsage `json:"usage"`
}

type EmbeddingData struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

type EmbeddingsUsage struct {
	TotalTokens int `json:"total_tokens"`
}

func NewEmbeddingsHandler(embCfg EmbeddingConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleEmbeddings(w, r, embCfg)
	}
}

func handleEmbeddings(w http.ResponseWriter, r *http.Request, embCfg EmbeddingConfig) {
	ctx := r.Context()

	var req EmbeddingsClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Input) == 0 {
		http.Error(w, "input must not be empty", http.StatusBadRequest)
		return
	}

	if len(req.Input) > maxEmbeddingBatchSize {
		http.Error(w, fmt.Sprintf("batch size %d exceeds maximum %d", len(req.Input), maxEmbeddingBatchSize), http.StatusBadRequest)
		return
	}

	estimatedTokens := estimateEmbeddingTokens(req.Input)
	if estimatedTokens > maxEmbeddingBatchTokens {
		http.Error(w, fmt.Sprintf("estimated tokens %d exceeds maximum %d", estimatedTokens, maxEmbeddingBatchTokens), http.StatusBadRequest)
		return
	}

	proxyReq := embeddingsProxyRequest{
		Input:      req.Input,
		Model:      embCfg.Model,
		Dimensions: embCfg.Dimensions,
	}

	start := time.Now()
	resp, err := proxyWithRetry(ctx, proxyReq, embCfg.Provider, "/embeddings")
	if err != nil {
		handleEmbeddingsProxyError(ctx, w, err)
		return
	}
	defer resp.Body.Close()

	if isErrorResponse(resp) {
		handleUpstreamError(ctx, w, resp)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read upstream response body", "component", "embeddings", "error", err)
		http.Error(w, "failed to read upstream response", http.StatusBadGateway)
		return
	}

	var embResp EmbeddingsResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		slog.WarnContext(ctx, "failed to parse upstream response", "component", "embeddings", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		return
	}

	durationMs := time.Since(start).Milliseconds()
	rec := buildEmbeddingCallRecord(embCfg.Model, embResp.Usage.TotalTokens, len(req.Input), embCfg.Pricing, durationMs)
	logCallRecord(ctx, rec)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func estimateEmbeddingTokens(input []string) int {
	total := 0
	for _, s := range input {
		total += len(s)
	}
	return total / charsPerToken
}

func handleEmbeddingsProxyError(ctx context.Context, w http.ResponseWriter, err error) {
	if errors.Is(err, errRateLimited) {
		slog.ErrorContext(ctx, "rate limit retries exhausted", "component", "embeddings")
		http.Error(w, "Rate limited after retries", http.StatusTooManyRequests)
		return
	}
	slog.ErrorContext(ctx, "upstream request failed", "component", "embeddings", "error", err)
	http.Error(w, "upstream request failed", http.StatusBadGateway)
}
