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

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages []Message `json:"messages"`
}

type StreamHandler struct {
	client       *openai.Client
	model        string
	systemPrompt string
	tools        []openai.Tool
}

func NewStreamHandler(apiKey, baseURL, model, systemPrompt string, tools []openai.Tool) StreamHandler {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	return StreamHandler{
		client:       openai.NewClientWithConfig(cfg),
		model:        model,
		systemPrompt: systemPrompt,
		tools:        tools,
	}
}

func (h StreamHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		http.Error(w, "messages required", http.StatusBadRequest)
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

	err := h.streamCompletion(r.Context(), req.Messages, w, flusher)
	if err != nil {
		slog.Error("stream error", "error", err)
	}
}

func (h StreamHandler) streamCompletion(ctx context.Context, messages []Message, w io.Writer, flusher http.Flusher) error {
	openaiMessages := buildMessages(h.systemPrompt, messages)

	stream, err := h.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    h.model,
		Messages: openaiMessages,
		Tools:    h.tools,
		Stream:   true,
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

func buildMessages(systemPrompt string, messages []Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	result = append(result, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})
	for _, m := range messages {
		result = append(result, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return result
}
