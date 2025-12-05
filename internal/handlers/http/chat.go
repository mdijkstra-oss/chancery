package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type StreamHandler struct {
	client *openai.Client
	model  string
}

func NewStreamHandler(apiKey, baseURL, model string) StreamHandler {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	return StreamHandler{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
	}
}

func (h StreamHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "message required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	err := h.streamCompletion(r.Context(), req.Message, w, flusher)
	if err != nil {
		slog.Error("stream error", "error", err)
	}
}

func (h StreamHandler) streamCompletion(ctx context.Context, message string, w io.Writer, flusher http.Flusher) error {
	stream, err := h.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model: h.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: message},
		},
		Stream: true,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		if len(resp.Choices) > 0 && resp.Choices[0].Delta.Content != "" {
			_, writeErr := w.Write([]byte(resp.Choices[0].Delta.Content))
			if writeErr != nil {
				return writeErr
			}
			flusher.Flush()
		}
	}
}
