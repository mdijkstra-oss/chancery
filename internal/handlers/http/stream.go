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

func streamWithUsageLogging(src io.Reader, dst io.Writer, flusher http.Flusher, pricing Pricing, verbose bool) {
	scanner := bufio.NewScanner(src)
	lineCount := 0
	var content strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		lineWithNewline := line + "\n"
		dst.Write([]byte(lineWithNewline))
		flusher.Flush()
		lineCount++

		if verbose {
			if delta := extractDeltaContent(line); delta != "" {
				content.WriteString(delta)
			}
		}

		if usage := extractUsage(line); usage != nil {
			logUsage(usage, pricing)
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

func extractDeltaContent(line string) string {
	data, ok := strings.CutPrefix(line, "data: ")
	if !ok || data == "[DONE]" {
		return ""
	}
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if json.Unmarshal([]byte(data), &chunk) == nil && len(chunk.Choices) > 0 {
		return chunk.Choices[0].Delta.Content
	}
	return ""
}
