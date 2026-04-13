package http

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

func isUnexpectedStreamError(err error) bool {
	return err != io.EOF && !errors.Is(err, context.Canceled)
}

var proxyHeaders = []string{"Content-Type", "Cache-Control"}

func copyHeaders(dst, src http.Header) {
	for _, h := range proxyHeaders {
		if v := src.Get(h); v != "" {
			dst.Set(h, v)
		}
	}
}

func forwardStream(ctx context.Context, src io.Reader, dst io.Writer, flusher http.Flusher, inspect bool, endpoint string) *UsageResponse {
	scanner := bufio.NewScanner(src)
	lineCount := 0
	var currentEvent string
	var completedData string

	for scanner.Scan() {
		line := scanner.Text()
		dst.Write([]byte(line + "\n"))
		flusher.Flush()
		lineCount++

		if eventType, ok := strings.CutPrefix(line, "event: "); ok {
			currentEvent = eventType
			continue
		}

		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}

		if currentEvent == "response.completed" {
			completedData = data
		}
	}

	if err := scanner.Err(); err != nil && isUnexpectedStreamError(err) {
		slog.ErrorContext(ctx, "stream read error", "component", "stream", "error", err)
		return nil
	}

	if inspect && completedData != "" {
		inspectRawJSON(endpoint+" response", []byte(completedData))
	}
	slog.DebugContext(ctx, "stream completed", "component", "stream", slog.Group("data", slog.Int("lines", lineCount)))
	return extractCompletedUsage(completedData)
}

func extractTextDelta(data string) string {
	var event TextDeltaEvent
	if json.Unmarshal([]byte(data), &event) == nil {
		return event.Delta
	}
	return ""
}

func extractCompletedUsage(data string) *UsageResponse {
	var event ResponseCompletedEvent
	if json.Unmarshal([]byte(data), &event) == nil {
		return event.Response.Usage
	}
	return nil
}
