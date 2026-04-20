package openai

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/providers/sse"
	"hermes-logos/internal/protocol"
	"hermes-logos/internal/ratelimit"
)

func Stream(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	req, err := BuildHTTPRequest(ctx, params, provider)
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
			return sse.StreamResult{}, ratelimit.Retryable(fmt.Errorf("openai returned status %d: %s", resp.StatusCode, body))
		}
		return sse.StreamResult{}, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, body)
	}

	sse.SetHeaders(w)
	sse.Flush(w)

	return relaySSE(ctx, w, bufio.NewScanner(resp.Body)), nil
}

func relaySSE(ctx context.Context, w io.Writer, scanner *bufio.Scanner) sse.StreamResult {
	var result sse.StreamResult
	var currentEvent string
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
			currentEvent = ""
		}
	}
	if err := scanner.Err(); err != nil {
		slog.ErrorContext(ctx, "stream scan error", "component", "openai", "error", err)
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
