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
)

const maxEmbeddingBatchSize = 512
const maxEmbeddingBatchTokens = 200_000
const maxEmbeddingRetries = 3
const charsPerToken = 4

type EmbeddingConfig struct {
	Model      string
	Dimensions int
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

func NewEmbeddingsHandler(cfg Config, embCfg EmbeddingConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleEmbeddings(w, r, cfg, embCfg)
	}
}

func handleEmbeddings(w http.ResponseWriter, r *http.Request, cfg Config, embCfg EmbeddingConfig) {
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

	resp, err := proxyEmbeddingsWithRetry(r.Context(), proxyReq, cfg)
	if err != nil {
		handleEmbeddingsProxyError(w, err)
		return
	}
	defer resp.Body.Close()

	if isErrorResponse(resp) {
		handleUpstreamError(w, resp)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("embeddings read error", "error", err)
		http.Error(w, "failed to read upstream response", http.StatusBadGateway)
		return
	}

	var embResp EmbeddingsResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		slog.Error("embeddings parse error", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		return
	}

	slog.Info("embeddings", "input_count", len(req.Input), "total_tokens", embResp.Usage.TotalTokens)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func proxyEmbeddingsRequest(ctx context.Context, req embeddingsProxyRequest, cfg Config) (*http.Response, error) {
	proxyReq, err := http.NewRequestWithContext(ctx, "POST", cfg.BaseURL+"/embeddings", jsonReader(req))
	if err != nil {
		return nil, err
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	return http.DefaultClient.Do(proxyReq)
}

func proxyEmbeddingsWithRetry(ctx context.Context, req embeddingsProxyRequest, cfg Config) (*http.Response, error) {
	for attempt := range maxEmbeddingRetries {
		resp, err := proxyEmbeddingsRequest(ctx, req, cfg)
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

		if attempt == maxEmbeddingRetries-1 {
			return nil, errRateLimited
		}

		delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
		slog.Warn("embeddings rate limited, retrying", "attempt", attempt+1, "delay", delay)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, errRateLimited
}

func estimateEmbeddingTokens(input []string) int {
	total := 0
	for _, s := range input {
		total += len(s)
	}
	return total / charsPerToken
}

func handleEmbeddingsProxyError(w http.ResponseWriter, err error) {
	if errors.Is(err, errQuotaExhausted) {
		slog.Error("embeddings quota exhausted")
		http.Error(w, "API quota exhausted", http.StatusPaymentRequired)
		return
	}
	if errors.Is(err, errRateLimited) {
		slog.Error("embeddings rate limited after retries")
		http.Error(w, "Rate limited after retries", http.StatusTooManyRequests)
		return
	}
	slog.Error("embeddings upstream request failed", "error", err)
	http.Error(w, "upstream request failed", http.StatusBadGateway)
}
