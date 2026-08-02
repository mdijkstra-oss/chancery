package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mdijkstra-oss/chancery/internal/auth"
	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/providers/openai"
	"github.com/mdijkstra-oss/chancery/internal/quota"
	"github.com/mdijkstra-oss/chancery/internal/ratelimit"
	"github.com/mdijkstra-oss/chancery/internal/telemetry"
	"github.com/mdijkstra-oss/chancery/internal/tokens"
)

const (
	maxEmbeddingBatchSize   = 512
	maxEmbeddingBatchTokens = 200_000
)

type EmbeddingsRequest struct {
	Input []string `json:"input"`
}

func NewEmbeddingsHandler(cfg prompts.PromptConfig, limiter *ratelimit.Limiter, quotaClient *quota.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleEmbeddings(w, r, cfg, limiter, quotaClient)
	}
}

func handleEmbeddings(w http.ResponseWriter, r *http.Request, cfg prompts.PromptConfig, limiter *ratelimit.Limiter, quotaClient *quota.Client) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	req, err := decodeEmbeddingsRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateEmbeddingsRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	quotaRequest := buildEmbeddingsQuotaRequest(RequestIDFromContext(ctx), auth.UserFromContext(ctx), req.Input, cfg)
	reservation, allowed := reserveQuota(ctx, w, quotaClient, quotaRequest)
	if !allowed {
		return
	}

	start := time.Now()
	res, err := ratelimit.Do(ctx, limiter, cfg.Model, 3, func() (openai.EmbedResult, error) {
		return openai.Embed(ctx, req.Input, cfg.Model, cfg.Dimensions, cfg.Provider)
	})
	duration := time.Since(start).Milliseconds()
	if err != nil {
		settleEmbeddingQuota(ctx, quotaClient, reservation, failedQuotaOutcome(ctx), 0)
		slog.ErrorContext(ctx, "embeddings upstream failed",
			"component", "embeddings",
			"error", err,
			slog.Group("data", slog.String("model", cfg.Model)),
		)
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}

	settleEmbeddingQuota(ctx, quotaClient, reservation, quota.OutcomeCompleted, res.TotalTokens)
	rec := telemetry.BuildEmbeddingCallRecord(cfg.Model, res.TotalTokens, len(req.Input), duration)
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
	if estimated := tokens.Estimate(req.Input); estimated > maxEmbeddingBatchTokens {
		return fmt.Errorf("estimated tokens %d exceeds maximum %d", estimated, maxEmbeddingBatchTokens)
	}
	return nil
}
