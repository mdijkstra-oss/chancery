package completions

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/providers/httpx"
	"hermes-logos/internal/providers/sse"
	"hermes-logos/internal/ratelimit"
)

func Stream(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	req, err := buildHTTPRequest(ctx, params, provider)
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("build request: %w", err)
	}
	resp, err := httpx.Client.Do(req)
	if err != nil {
		if httpx.IsConnectTimeout(err) {
			return sse.StreamResult{}, ratelimit.Retryable(fmt.Errorf("completions: connect timeout: %w", err))
		}
		return sse.StreamResult{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusTooManyRequests {
			return sse.StreamResult{}, ratelimit.Retryable(fmt.Errorf("completions returned status %d: %s", resp.StatusCode, body))
		}
		return sse.StreamResult{}, fmt.Errorf("completions returned status %d: %s", resp.StatusCode, body)
	}

	sse.SetHeaders(w)
	sse.Flush(w)

	body := httpx.WithStallTimeout(resp.Body, httpx.StallTimeout)
	defer body.Close()

	state := &EmitState{}
	var lastUsage *protocol.UsageResponse
	var lastFinishReason string
	var streamErr error

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		raw := []byte(data)

		if reason := ExtractFinishReason(raw); reason != "" {
			lastFinishReason = reason
		}

		usage := ExtractUsage(raw)
		if usage != nil {
			lastUsage = usage
		}

		for _, event := range ChunkToEvents(raw, state) {
			sse.WriteEvent(w, event.Type, event.Data)
		}
		sse.Flush(w)
	}
	if err := scanner.Err(); err != nil {
		slog.ErrorContext(ctx, "completions stream scan error",
			"component", "completions",
			"error", err,
			"model", params.Model,
		)
		streamErr = err
	}

	for _, event := range FlushReasoning(state) {
		sse.WriteEvent(w, event.Type, event.Data)
	}
	for _, event := range FlushActiveCalls(state) {
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

	return sse.StreamResult{Usage: lastUsage}, nil
}

func buildHTTPRequest(ctx context.Context, params protocol.RequestParams, provider prompts.ProviderConfig) (*http.Request, error) {
	reqBody, err := BuildRequest(params, provider.Strict)
	if err != nil {
		return nil, fmt.Errorf("build completions request: %w", err)
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	url := provider.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	return req, nil
}
