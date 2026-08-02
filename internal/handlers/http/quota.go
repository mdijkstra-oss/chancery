package http

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/protocol"
	"github.com/mdijkstra-oss/chancery/internal/quota"
	"github.com/mdijkstra-oss/chancery/internal/tokens"
)

const (
	chatOperation       = "chat"
	embeddingsOperation = "embeddings"
)

func buildChatQuotaRequest(requestID, subject, endpoint string, params protocol.RequestParams, config prompts.PromptConfig) quota.ReserveRequest {
	return quota.ReserveRequest{
		RequestID:            requestID,
		Subject:              subject,
		Operation:            chatOperation,
		Endpoint:             endpoint,
		Provider:             config.Provider.Key,
		Model:                params.Model,
		ServiceTier:          params.ServiceTier,
		ReasoningEffort:      params.ReasoningEffort,
		EstimatedInputTokens: estimateChatInputTokens(params),
		MaximumOutputTokens:  params.MaxTokens,
	}
}

func buildEmbeddingsQuotaRequest(requestID, subject string, input []string, config prompts.PromptConfig) quota.ReserveRequest {
	return quota.ReserveRequest{
		RequestID:            requestID,
		Subject:              subject,
		Operation:            embeddingsOperation,
		Endpoint:             embeddingsOperation,
		Provider:             config.Provider.Key,
		Model:                config.Model,
		EstimatedInputTokens: tokens.Estimate(input),
	}
}

func reserveQuota(ctx context.Context, w http.ResponseWriter, client *quota.Client, request quota.ReserveRequest) (quota.Reservation, bool) {
	reservation, err := client.Reserve(ctx, request)
	if err != nil {
		slog.ErrorContext(ctx, "quota reservation failed", "component", "quota", "error", err)
		http.Error(w, "quota service unavailable", http.StatusServiceUnavailable)
		return quota.Reservation{}, false
	}
	if reservation.Allowed {
		return reservation, true
	}
	if reservation.RetryAfterSeconds > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(reservation.RetryAfterSeconds))
	}
	slog.InfoContext(ctx, "quota denied",
		"component", "quota",
		slog.Group("data",
			slog.String("reason", reservation.Reason),
			slog.String("operation", request.Operation),
			slog.String("model", request.Model),
		),
	)
	http.Error(w, "quota exceeded", http.StatusTooManyRequests)
	return quota.Reservation{}, false
}

func settleQuota(ctx context.Context, client *quota.Client, reservation quota.Reservation, outcome quota.Outcome, usage *protocol.UsageResponse) {
	settlement := quota.Settlement{
		ReservationID: reservation.ID,
		Outcome:       outcome,
		Usage:         quotaUsage(usage),
	}
	if err := client.Settle(ctx, settlement); err != nil {
		slog.ErrorContext(ctx, "quota settlement failed", "component", "quota", "error", err)
	}
}

func settleEmbeddingQuota(ctx context.Context, client *quota.Client, reservation quota.Reservation, outcome quota.Outcome, totalTokens int) {
	var usage *quota.Usage
	if totalTokens > 0 {
		usage = &quota.Usage{InputTokens: totalTokens, TotalTokens: totalTokens}
	}
	settlement := quota.Settlement{
		ReservationID: reservation.ID,
		Outcome:       outcome,
		Usage:         usage,
	}
	if err := client.Settle(ctx, settlement); err != nil {
		slog.ErrorContext(ctx, "quota settlement failed", "component", "quota", "error", err)
	}
}

func quotaUsage(usage *protocol.UsageResponse) *quota.Usage {
	if usage == nil {
		return nil
	}
	return &quota.Usage{
		InputTokens:       usage.InputTokens,
		CachedInputTokens: protocol.CachedInputTokens(usage),
		CacheWriteTokens:  protocol.CacheWriteTokens(usage),
		OutputTokens:      usage.OutputTokens,
		ReasoningTokens:   protocol.ReasoningTokens(usage),
		TotalTokens:       usage.TotalTokens,
	}
}

func failedQuotaOutcome(ctx context.Context) quota.Outcome {
	if ctx.Err() != nil {
		return quota.OutcomeCancelled
	}
	return quota.OutcomeFailed
}

func estimateChatInputTokens(params protocol.RequestParams) int {
	byteCount := len(params.SystemPrompt) + len(params.ResponseFormat)
	for _, message := range params.Messages {
		byteCount += len(message)
	}
	for _, tool := range params.Tools {
		byteCount += len(tool)
	}
	return tokens.EstimateByteCount(byteCount)
}
