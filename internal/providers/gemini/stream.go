package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/genai"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/providers/httpx"
	"hermes-logos/internal/providers/sse"
	"hermes-logos/internal/ratelimit"
)

var globalCacheStore = NewCacheStore()

func Stream(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:     provider.APIKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: httpx.Client,
	})
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("create gemini client: %w", err)
	}

	leadingSystem, rest := ExtractLeadingSystem(params.Messages)
	callIDMap := BuildCallIDToName(params.Messages)
	isThinking := params.ReasoningEffort != "" && params.ReasoningEffort != "off"

	config := BuildConfig(params, leadingSystem)

	cachedName, isFirstRead := resolveCache(ctx, client, params, config)

	messagesToConvert := rest
	if cachedName != "" {
		_, tail := SplitAtLastBreakpoint(rest, params.CacheBreakpoints)
		messagesToConvert = tail
		config.CachedContent = cachedName
		config.SystemInstruction = nil
		config.Tools = nil
		config.ToolConfig = nil
	}

	contents := EnsureThoughtSignatures(
		MergeConsecutiveContents(MessagesToContents(messagesToConvert, callIDMap)),
		isThinking,
	)

	headersWritten := false
	state := &EmitState{}
	var lastUsage *protocol.UsageResponse
	var lastFinishReason genai.FinishReason
	var streamErr error

	for chunk, err := range client.Models.GenerateContentStream(ctx, params.Model, contents, config) {
		if err != nil {
			if !headersWritten && isRateLimitError(err) {
				delay := ExtractRetryDelay(err)
				if delay > 0 {
					return sse.StreamResult{}, ratelimit.RetryableWithDelay(err, delay)
				}
				return sse.StreamResult{}, ratelimit.Retryable(err)
			}
			if !headersWritten && httpx.IsConnectTimeout(err) {
				return sse.StreamResult{}, ratelimit.Retryable(fmt.Errorf("gemini: connect timeout: %w", err))
			}
			if !headersWritten {
				return sse.StreamResult{}, fmt.Errorf("gemini stream: %w", err)
			}
			slog.ErrorContext(ctx, "gemini stream chunk error", "component", "gemini", "error", err)
			streamErr = err
			break
		}
		if !headersWritten {
			sse.SetHeaders(w)
			sse.Flush(w)
			headersWritten = true
		}
		if feedback := ExtractPromptFeedback(chunk); feedback != "" {
			event := BuildTextDeltaEvent(feedback)
			sse.WriteEvent(w, event.Type, event.Data)
		}
		if reason := ExtractFinishReason(chunk); reason != "" {
			lastFinishReason = reason
		}
		usage := ExtractGeminiUsage(chunk)
		if usage != nil {
			lastUsage = usage
		}
		for _, event := range ChunkToEvents(chunk, state) {
			sse.WriteEvent(w, event.Type, event.Data)
		}
		sse.Flush(w)
	}

	if !headersWritten {
		sse.SetHeaders(w)
		sse.Flush(w)
	}

	for _, event := range flushThought(state) {
		sse.WriteEvent(w, event.Type, event.Data)
	}

	if streamErr != nil {
		event := sse.BuildFailedEvent("stream_error", streamErr.Error())
		sse.WriteEvent(w, event.Type, event.Data)
	} else if event := FinishReasonToEvent(lastFinishReason); event != nil {
		sse.WriteEvent(w, event.Type, event.Data)
	}

	completed := sse.BuildCompletedEvent(lastUsage)
	sse.WriteEvent(w, completed.Type, completed.Data)
	sse.Flush(w)

	if isFirstRead {
		lastUsage = addCacheCreationCost(lastUsage)
	}

	return sse.StreamResult{Usage: lastUsage}, nil
}

func hasCacheBreakpoints(params protocol.RequestParams) bool {
	return params.CacheTTL > 0 && len(params.CacheBreakpoints) > 0
}

func resolveCache(ctx context.Context, client *genai.Client, params protocol.RequestParams, config *genai.GenerateContentConfig) (string, bool) {
	if !hasCacheBreakpoints(params) {
		return "", false
	}

	prefix, _ := SplitAtLastBreakpoint(params.Messages, params.CacheBreakpoints)
	if len(prefix) == 0 {
		return "", false
	}

	ttl := time.Duration(params.CacheTTL) * time.Second
	hash := ContentHash(params.Model, params.SystemPrompt, params.Tools, prefix)
	now := time.Now()

	entry, ok := globalCacheStore.FindValid(hash, now)
	if ok {
		if ShouldRenew(entry, now) {
			renewCacheTTL(ctx, client, entry.ResourceName, ttl, hash)
		}
		isFirstRead := globalCacheStore.MarkFirstRead(hash)
		slog.InfoContext(ctx, "gemini cache hit", "component", "gemini", "resource", entry.ResourceName, "first_read", isFirstRead)
		return entry.ResourceName, isFirstRead
	}

	name := createOrWaitCache(ctx, client, params, config, prefix, hash, ttl)
	if name == "" {
		return "", false
	}
	isFirstRead := globalCacheStore.MarkFirstRead(hash)
	return name, isFirstRead
}

func createOrWaitCache(ctx context.Context, client *genai.Client, params protocol.RequestParams, config *genai.GenerateContentConfig, prefix []json.RawMessage, hash string, ttl time.Duration) string {
	isCreator, fl := globalCacheStore.AcquireOrWait(hash)
	if !isCreator {
		entry, err := WaitInflight(fl)
		if err != nil {
			slog.WarnContext(ctx, "gemini cache inflight failed", "component", "gemini", "error", err)
			return ""
		}
		slog.InfoContext(ctx, "gemini cache hit (waited)", "component", "gemini", "resource", entry.ResourceName)
		return entry.ResourceName
	}

	entry, err := createCachedContent(ctx, client, params, config, prefix, ttl)
	globalCacheStore.CompleteInflight(hash, entry, err)
	if err != nil {
		slog.WarnContext(ctx, "gemini cache create failed", "component", "gemini", "error", err)
		return ""
	}
	slog.InfoContext(ctx, "gemini cache created", "component", "gemini", "resource", entry.ResourceName)
	return entry.ResourceName
}

func createCachedContent(ctx context.Context, client *genai.Client, params protocol.RequestParams, config *genai.GenerateContentConfig, prefix []json.RawMessage, ttl time.Duration) (CacheEntry, error) {
	callIDMap := BuildCallIDToName(params.Messages)
	prefixContents := MergeConsecutiveContents(MessagesToContents(StripLeadingSystem(prefix), callIDMap))

	createCfg := &genai.CreateCachedContentConfig{
		TTL:      ttl,
		Contents: prefixContents,
	}
	if config.SystemInstruction != nil {
		createCfg.SystemInstruction = config.SystemInstruction
	}
	if config.Tools != nil {
		createCfg.Tools = config.Tools
	}
	if config.ToolConfig != nil {
		createCfg.ToolConfig = config.ToolConfig
	}

	cached, err := client.Caches.Create(ctx, params.Model, createCfg)
	if err != nil {
		return CacheEntry{}, fmt.Errorf("create cached content: %w", err)
	}

	return CacheEntry{
		ResourceName: cached.Name,
		ExpireTime:   cached.ExpireTime,
		TTL:          ttl,
	}, nil
}

func renewCacheTTL(ctx context.Context, client *genai.Client, resourceName string, ttl time.Duration, hash string) {
	updated, err := client.Caches.Update(ctx, resourceName, &genai.UpdateCachedContentConfig{
		TTL: ttl,
	})
	if err != nil {
		slog.WarnContext(ctx, "gemini cache renew failed", "component", "gemini", "resource", resourceName, "error", err)
		return
	}
	globalCacheStore.UpdateExpiry(hash, updated.ExpireTime)
	slog.InfoContext(ctx, "gemini cache renewed", "component", "gemini", "resource", resourceName)
}

func addCacheCreationCost(usage *protocol.UsageResponse) *protocol.UsageResponse {
	if usage == nil || usage.InputTokensDetails == nil || usage.InputTokensDetails.CachedTokens == 0 {
		return usage
	}
	cached := usage.InputTokensDetails.CachedTokens
	adjusted := *usage
	adjusted.InputTokens += cached
	adjusted.TotalTokens += cached
	details := *adjusted.InputTokensDetails
	details.CacheCreationTokens = cached
	adjusted.InputTokensDetails = &details
	return &adjusted
}

func isRateLimitError(err error) bool {
	var apiErr *genai.APIError
	return errors.As(err, &apiErr) && apiErr.Code == http.StatusTooManyRequests
}
