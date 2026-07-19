package openai

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/matthijn/hermes-logos/internal/prompts"
	"github.com/matthijn/hermes-logos/internal/providers/httpx"
	"github.com/matthijn/hermes-logos/internal/providers/sse"
	"github.com/matthijn/hermes-logos/internal/protocol"
	"github.com/matthijn/hermes-logos/internal/ratelimit"
)

func Stream(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	req, err := BuildHTTPRequest(ctx, params, provider)
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("build request: %w", err)
	}
	resp, err := httpx.Client.Do(req)
	if err != nil {
		if httpx.IsConnectTimeout(err) {
			return sse.StreamResult{}, ratelimit.Retryable(fmt.Errorf("openai: connect timeout: %w", err))
		}
		return sse.StreamResult{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusTooManyRequests {
			err := fmt.Errorf("openai returned status %d: %s", resp.StatusCode, body)
			if d := ratelimit.ParseRetryAfterHeader(resp.Header.Get("Retry-After")); d > 0 {
				return sse.StreamResult{}, ratelimit.RetryableWithDelay(err, d)
			}
			return sse.StreamResult{}, ratelimit.Retryable(err)
		}
		return sse.StreamResult{}, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, body)
	}

	sse.SetHeaders(w)
	sse.Flush(w)

	body := httpx.WithStallTimeout(resp.Body, httpx.StallTimeout)
	defer body.Close()
	return relaySSE(ctx, w, bufio.NewScanner(body), params.Model), nil
}

func relaySSE(ctx context.Context, w io.Writer, scanner *bufio.Scanner, model string) sse.StreamResult {
	var result sse.StreamResult
	var currentEvent string
	var eventsRelayed int
	var lastEvent string
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(w, line)
		if isEventLine(line) {
			currentEvent = strings.TrimPrefix(line, "event: ")
		}
		if isDataLine(line) && currentEvent == "response.completed" {
			data := strings.TrimPrefix(line, "data: ")
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

func isEventLine(line string) bool {
	return strings.HasPrefix(line, "event: ")
}

func isDataLine(line string) bool {
	return strings.HasPrefix(line, "data: ")
}

func isEventBoundary(line string) bool {
	return line == ""
}
