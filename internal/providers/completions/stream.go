package completions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/protocol"
	"github.com/mdijkstra-oss/chancery/internal/providers/httpstream"
	"github.com/mdijkstra-oss/chancery/internal/providers/sse"
)

func Stream(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	req, err := buildHTTPRequest(ctx, params, provider)
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("build request: %w", err)
	}
	scanner, body, err := httpstream.Open(w, req, "completions")
	if err != nil {
		return sse.StreamResult{}, err
	}
	defer body.Close()

	state := &EmitState{}
	var lastUsage *protocol.UsageResponse
	var lastFinishReason string
	var streamErr error

	for scanner.Scan() {
		line := scanner.Text()
		data, ok := sse.DataField(line)
		if !ok {
			continue
		}
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
