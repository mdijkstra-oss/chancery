package openai

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/matthijn/hermes-logos/internal/prompts"
	"github.com/matthijn/hermes-logos/internal/protocol"
	"github.com/matthijn/hermes-logos/internal/providers/httpstream"
	"github.com/matthijn/hermes-logos/internal/providers/sse"
)

func Stream(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	req, err := BuildHTTPRequest(ctx, params, provider)
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("build request: %w", err)
	}
	scanner, body, err := httpstream.Open(w, req, "openai")
	if err != nil {
		return sse.StreamResult{}, err
	}
	defer body.Close()
	return relaySSE(ctx, w, scanner, params.Model), nil
}

func relaySSE(ctx context.Context, w io.Writer, scanner *bufio.Scanner, model string) sse.StreamResult {
	var result sse.StreamResult
	var currentEvent string
	var eventsRelayed int
	var lastEvent string
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(w, line)
		if event, ok := sse.EventField(line); ok {
			currentEvent = event
		}
		if data, ok := sse.DataField(line); ok && currentEvent == "response.completed" {
			result.Usage = ExtractUsageFromCompleted([]byte(data))
		}
		if isEventBoundary(line) {
			sse.Flush(w)
			eventsRelayed++
			lastEvent = currentEvent
			currentEvent = ""
		}
	}
	if err := scanner.Err(); err != nil {
		slog.ErrorContext(ctx, "stream scan error",
			"component", "openai",
			"error", err,
			"model", model,
			"events_relayed", eventsRelayed,
			"last_event", lastEvent,
		)
	}
	return result
}

func isEventBoundary(line string) bool {
	return line == ""
}
