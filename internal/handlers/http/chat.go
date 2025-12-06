package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/sashabaranov/go-openai"
	"hermes-logos/internal/lib/utils"
	"hermes-logos/internal/parser"
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
}

func NewStreamHandler(apiKey, baseURL, model, systemPrompt string) StreamHandler {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	return StreamHandler{
		client:       openai.NewClientWithConfig(cfg),
		model:        model,
		systemPrompt: systemPrompt,
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

	meta := buildRequestMeta(h.model)
	err := h.streamCompletion(r.Context(), req.Messages, w, flusher, meta)
	if err != nil {
		slog.Error("stream error", "error", err)
	}
}

func (h StreamHandler) streamCompletion(ctx context.Context, messages []Message, w io.Writer, flusher http.Flusher, meta parser.RequestMeta) error {
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: h.systemPrompt,
	})
	for _, m := range messages {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	stream, err := h.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    h.model,
		Messages: openaiMessages,
		Stream:   true,
	})
	if err != nil {
		return err
	}
	defer stream.Close()

	state := parser.NewState()

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		if len(resp.Choices) > 0 && resp.Choices[0].Delta.Content != "" {
			chunk := resp.Choices[0].Delta.Content
			result := parser.Process(state, chunk, meta)
			state = result.State

			if result.Output != "" {
				_, writeErr := w.Write([]byte(result.Output))
				if writeErr != nil {
					return writeErr
				}
				flusher.Flush()
			}
		}
	}
}

func buildRequestMeta(model string) parser.RequestMeta {
	return parser.RequestMeta{
		Model:     model,
		RequestID: utils.GenerateID(),
	}
}
