package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/providers/openai"
	"hermes-logos/internal/ratelimit"
	"hermes-logos/internal/telemetry"
)

const (
	maxEmbeddingBatchSize   = 512
	maxEmbeddingBatchTokens = 200_000
	charsPerToken           = 4
)

type EmbeddingsRequest struct {
	Input []string `json:"input"`
}

func NewEmbeddingsHandler(cfg prompts.PromptConfig, limiter *ratelimit.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleEmbeddings(w, r, cfg, limiter)
	}
}

func handleEmbeddings(w http.ResponseWriter, r *http.Request, cfg prompts.PromptConfig, limiter *ratelimit.Limiter) {
	ctx := r.Context()

	req, err := decodeEmbeddingsRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateEmbeddingsRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	start := time.Now()
	res, err := ratelimit.Do(ctx, limiter, cfg.Provider.Key, 3, func() (openai.EmbedResult, error) {
		return openai.Embed(ctx, req.Input, cfg.Model, cfg.Dimensions, cfg.Provider)
	})
	duration := time.Since(start).Milliseconds()
	if err != nil {
		slog.ErrorContext(ctx, "embeddings upstream failed",
			"component", "embeddings",
			"error", err,
			slog.Group("data", slog.String("model", cfg.Model)),
		)
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}

	rec := telemetry.BuildEmbeddingCallRecord(cfg.Model, res.TotalTokens, len(req.Input), cfg.Pricing, duration)
	telemetry.LogCallRecord(ctx, rec)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res.Body)
}

func decodeEmbeddingsRequest(r *http.Request) (EmbeddingsRequest, error) {
	var req EmbeddingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, fmt.Errorf("invalid request body")
	}
	return req, nil
}

func validateEmbeddingsRequest(req EmbeddingsRequest) error {
	if len(req.Input) == 0 {
		return fmt.Errorf("input must not be empty")
	}
	if len(req.Input) > maxEmbeddingBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum %d", len(req.Input), maxEmbeddingBatchSize)
	}
	if tokens := estimateTokens(req.Input); tokens > maxEmbeddingBatchTokens {
		return fmt.Errorf("estimated tokens %d exceeds maximum %d", tokens, maxEmbeddingBatchTokens)
	}
	return nil
}

func estimateTokens(input []string) int {
	total := 0
	for _, s := range input {
		total += len(s)
	}
	return total / charsPerToken
}
