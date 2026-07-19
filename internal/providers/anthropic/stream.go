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
	"github.com/matthijn/hermes-logos/internal/providers/httpstream"
	"github.com/matthijn/hermes-logos/internal/providers/sse"
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

	scanner, stream, err := httpstream.Open(w, httpReq, "anthropic")
	if err != nil {
		return sse.StreamResult{}, err
	}
	defer stream.Close()

	state := &EmitState{}
	var lastUsage *protocol.UsageResponse
	var lastMessageDeltaData []byte
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
