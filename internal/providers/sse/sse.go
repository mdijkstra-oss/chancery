package sse

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
)

type StreamResult struct {
	Usage *protocol.UsageResponse
}

type StreamFunc func(ctx context.Context, w io.Writer, params protocol.RequestParams, provider prompts.ProviderConfig) (StreamResult, error)

func SetHeaders(w io.Writer) {
	if hw, ok := w.(http.ResponseWriter); ok {
		hw.Header().Set("Content-Type", "text/event-stream")
		hw.Header().Set("Cache-Control", "no-cache")
		hw.Header().Set("Connection", "keep-alive")
	}
}

func WriteEvent(w io.Writer, eventType, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
}

func Flush(w io.Writer) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
