package gemini

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"google.golang.org/genai"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/providers/sse"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/ratelimit"
)

func Stream(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  provider.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("create gemini client: %w", err)
	}

	leadingSystem, rest := ExtractLeadingSystem(params.Messages)
	callIDMap := BuildCallIDToName(params.Messages)
	isThinking := params.ReasoningEffort != "" && params.ReasoningEffort != "off"
	contents := EnsureThoughtSignatures(
		MergeConsecutiveContents(MessagesToContents(rest, callIDMap)),
		isThinking,
	)
	config := BuildConfig(params, leadingSystem)

	headersWritten := false
	state := &EmitState{}
	var lastUsage *protocol.UsageResponse
	var lastFinishReason genai.FinishReason
	var streamErr error

	for chunk, err := range client.Models.GenerateContentStream(ctx, params.Model, contents, config) {
		if err != nil {
			if !headersWritten && isRateLimitError(err) {
				return sse.StreamResult{}, ratelimit.Retryable(err)
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
		event := BuildFailedEvent("stream_error", streamErr.Error())
		sse.WriteEvent(w, event.Type, event.Data)
	} else if event := FinishReasonToEvent(lastFinishReason); event != nil {
		sse.WriteEvent(w, event.Type, event.Data)
	}

	completed := BuildCompletedEvent(lastUsage)
	sse.WriteEvent(w, completed.Type, completed.Data)
	sse.Flush(w)

	return sse.StreamResult{Usage: lastUsage}, nil
}

func isRateLimitError(err error) bool {
	var apiErr *genai.APIError
	return errors.As(err, &apiErr) && apiErr.Code == http.StatusTooManyRequests
}
