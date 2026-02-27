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

	"hermes-logos/internal/prompts"
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

func streamWithUsageLogging(src io.Reader, dst io.Writer, flusher http.Flusher, cfg Config, endpoint string, pricing prompts.Pricing, reasoningEffort string, estimatedTokens int) {
	scanner := bufio.NewScanner(src)
	lineCount := 0
	var currentEvent string
	var completedData string
	var completedUsage *UsageResponse

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
			completedUsage = extractCompletedUsage(data)
		}
	}

	if err := scanner.Err(); err != nil && isUnexpectedStreamError(err) {
		slog.Error("stream read error", "error", err)
	} else {
		if cfg.Inspect && completedData != "" {
			inspectRawJSON(endpoint+" response", []byte(completedData))
		}
		if completedUsage != nil {
			logUsage(endpoint, completedUsage, pricing, reasoningEffort, estimatedTokens)
		}
		slog.Info("stream_complete", "lines_received", lineCount)
	}
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
