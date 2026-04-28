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
	"hermes-logos/internal/providers/sse"
	"hermes-logos/internal/ratelimit"
)

func Stream(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	req, err := buildHTTPRequest(ctx, params, provider)
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
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

	state := &EmitState{}
	var lastUsage *protocol.UsageResponse
	var streamErr error

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		usage := ExtractUsage([]byte(data))
		if usage != nil {
			lastUsage = usage
		}

		for _, event := range ChunkToEvents([]byte(data), state) {
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

	for _, event := range FlushActiveCalls(state) {
		sse.WriteEvent(w, event.Type, event.Data)
	}

	if streamErr != nil {
		event := BuildFailedEvent("stream_error", streamErr.Error())
		sse.WriteEvent(w, event.Type, event.Data)
	}

	completed := BuildCompletedEvent(lastUsage)
	sse.WriteEvent(w, completed.Type, completed.Data)
	sse.Flush(w)

	return sse.StreamResult{Usage: lastUsage}, nil
}

func buildHTTPRequest(ctx context.Context, params protocol.RequestParams, provider prompts.ProviderConfig) (*http.Request, error) {
	body, err := json.Marshal(BuildRequest(params, provider.Strict))
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
