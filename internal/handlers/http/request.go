package http

import (
	"encoding/json"
	"io"

	"github.com/sashabaranov/go-openai"
)

func prependSystemMessage(systemPrompt string, messages []json.RawMessage) []json.RawMessage {
	sysMsg := SystemMessage{Role: "system", Content: systemPrompt}
	sysMsgJSON, _ := json.Marshal(sysMsg)
	return append([]json.RawMessage{sysMsgJSON}, messages...)
}

func buildOpenAIRequest(model, provider, systemPrompt string, tools []openai.Tool, messages []json.RawMessage, includeReasoning, enableCaching bool) OpenAIRequest {
	req := OpenAIRequest{
		Model:    model,
		Messages: prependSystemMessage(systemPrompt, messages),
		Tools:    wrapToolsWithCache(tools, enableCaching),
		Stream:   true,
		Usage:    &UsageRequest{Include: true},
	}
	if provider != "" {
		req.Provider = &ProviderPreference{Only: []string{provider}}
	}
	req.IncludeReasoning = &includeReasoning
	return req
}

func encodeJSON(v any, pw *io.PipeWriter) {
	pw.CloseWithError(json.NewEncoder(pw).Encode(v))
}

func jsonReader(v any) io.Reader {
	pr, pw := io.Pipe()
	go encodeJSON(v, pw)
	return pr
}
