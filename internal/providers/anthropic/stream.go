package anthropic

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

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("anthropic: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return sse.StreamResult{}, ratelimit.Retryable(fmt.Errorf("anthropic: rate limited (429)"))
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return sse.StreamResult{}, fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	sse.SetHeaders(w)
	sse.Flush(w)

	state := &EmitState{}
	var lastUsage *protocol.UsageResponse
	var lastMessageDeltaData []byte

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	var currentEventType string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := []byte(strings.TrimPrefix(line, "data: "))

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
