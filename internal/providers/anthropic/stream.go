package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/matthijn/hermes-logos/internal/prompts"
	"github.com/matthijn/hermes-logos/internal/protocol"
	"github.com/matthijn/hermes-logos/internal/providers/httpx"
	"github.com/matthijn/hermes-logos/internal/providers/sse"
	"github.com/matthijn/hermes-logos/internal/ratelimit"
)

func Stream(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	req := BuildRequest(params, provider)
	body, err := json.Marshal(req)
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("anthropic: create request: %w", err)
	}
	httpReq.Header.Set("x-api-key", provider.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := httpx.Client.Do(httpReq)
	if err != nil {
		if httpx.IsConnectTimeout(err) {
			return sse.StreamResult{}, ratelimit.Retryable(fmt.Errorf("anthropic: connect timeout: %w", err))
		}
		return sse.StreamResult{}, fmt.Errorf("anthropic: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		err := fmt.Errorf("anthropic: rate limited (429)")
		if d := ratelimit.ParseRetryAfterHeader(resp.Header.Get("Retry-After")); d > 0 {
			return sse.StreamResult{}, ratelimit.RetryableWithDelay(err, d)
		}
		return sse.StreamResult{}, ratelimit.Retryable(err)
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return sse.StreamResult{}, fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	sse.SetHeaders(w)
	sse.Flush(w)

	stream := httpx.WithStallTimeout(resp.Body, httpx.StallTimeout)
	defer stream.Close()

	state := &EmitState{}
	var lastUsage *protocol.UsageResponse
	var lastMessageDeltaData []byte

	scanner := sse.NewScanner(stream)
	var currentEventType string

	for scanner.Scan() {
		line := scanner.Text()

		if eventType, ok := sse.EventField(line); ok {
			currentEventType = eventType
			continue
		}

		if value, ok := sse.DataField(line); ok {
			data := []byte(value)

			if currentEventType == "message_delta" {
				lastMessageDeltaData = data
			}

			events := HandleEvent(currentEventType, data, state)
			for _, ev := range events {
				sse.WriteEvent(w, ev.Type, ev.Data)
			}
			sse.Flush(w)
			currentEventType = ""
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		slog.ErrorContext(ctx, "anthropic stream scan error", "component", "anthropic", "error", err)
		event := sse.BuildFailedEvent("stream_error", err.Error())
		sse.WriteEvent(w, event.Type, event.Data)
	}

	lastUsage = ExtractUsage(state, lastMessageDeltaData)
	completed := sse.BuildCompletedEvent(lastUsage)
	sse.WriteEvent(w, completed.Type, completed.Data)
	sse.Flush(w)

	return sse.StreamResult{Usage: lastUsage}, nil
}
