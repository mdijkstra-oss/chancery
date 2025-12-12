package http

import (
	"bufio"
	"context"
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
	var collected strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		lineWithNewline := line + "\n"
		dst.Write([]byte(lineWithNewline))
		flusher.Flush()
		lineCount++

		if verbose {
			collected.WriteString(lineWithNewline)
		}

		if usage := extractUsage(line); usage != nil {
			logUsage(usage, pricing)
		}
	}

	if err := scanner.Err(); err != nil && isUnexpectedStreamError(err) {
		slog.Error("stream read error", "error", err)
	} else {
		slog.Info("stream_complete", "lines_received", lineCount)
		if verbose {
			slog.Info("raw_response", "data", collected.String())
		}
	}
}
