package gemini

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"google.golang.org/genai"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/providers/sse"
	"hermes-logos/internal/protocol"
)

func Stream(ctx context.Context, w http.ResponseWriter, params protocol.RequestParams, provider prompts.ProviderConfig) (sse.StreamResult, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  provider.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return sse.StreamResult{}, fmt.Errorf("create gemini client: %w", err)
	}

	leadingSystem, rest := ExtractLeadingSystem(params.Messages)
	callIDMap := BuildCallIDToName(params.Messages)
	contents := MergeConsecutiveContents(MessagesToContents(rest, callIDMap))
	config := BuildConfig(params, leadingSystem)

	sse.SetHeaders(w)
	sse.Flush(w)

	state := &EmitState{}
	var lastUsage *protocol.UsageResponse

	for chunk, err := range client.Models.GenerateContentStream(ctx, params.Model, contents, config) {
		if err != nil {
			slog.ErrorContext(ctx, "gemini stream chunk error", "component", "gemini", "error", err)
			break
		}
		usage := ExtractGeminiUsage(chunk)
		if usage != nil {
			lastUsage = usage
		}
		for _, event := range ChunkToEvents(chunk, state) {
			sse.WriteEvent(w, event.Type, event.Data)
		}
		sse.Flush(w)
	}

	flushEvents := flushThought(state)
	for _, event := range flushEvents {
		sse.WriteEvent(w, event.Type, event.Data)
	}

	completed := BuildCompletedEvent(lastUsage)
	sse.WriteEvent(w, completed.Type, completed.Data)
	sse.Flush(w)

	return sse.StreamResult{Usage: lastUsage}, nil
}
