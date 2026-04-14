package sse

import (
	"context"
	"fmt"
	"net/http"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
)

type StreamResult struct {
	Usage *protocol.UsageResponse
}

type StreamFunc func(ctx context.Context, w http.ResponseWriter, params protocol.RequestParams, provider prompts.ProviderConfig) (StreamResult, error)

func SetHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

func WriteEvent(w http.ResponseWriter, eventType, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
}

func Flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
