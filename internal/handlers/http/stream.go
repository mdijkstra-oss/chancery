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

func streamWithUsageLogging(src io.Reader, dst io.Writer, flusher http.Flusher, verbose bool, endpoint string, toolNames []string, pricing prompts.Pricing) {
	scanner := bufio.NewScanner(src)
	lineCount := 0
	var content strings.Builder
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()
		lineWithNewline := line + "\n"
		dst.Write([]byte(lineWithNewline))
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

		if verbose && currentEvent == "response.output_text.delta" {
			if delta := extractTextDelta(data); delta != "" {
				content.WriteString(delta)
			}
		}

		if currentEvent == "response.completed" {
			if usage := extractCompletedUsage(data); usage != nil {
				logUsage(endpoint, toolNames, usage, pricing)
			}
		}
	}

	if err := scanner.Err(); err != nil && isUnexpectedStreamError(err) {
		slog.Error("stream read error", "error", err)
	} else {
		slog.Info("stream_complete", "lines_received", lineCount)
		if verbose && content.Len() > 0 {
			slog.Info("raw_response", "content", content.String())
		}
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
