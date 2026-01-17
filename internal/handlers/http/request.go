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

// TODO: prompt cache retention (only certain models)
func buildOpenAIRequest(model, systemPrompt, gptVerbosity, reasoningEffort string, tools []openai.Tool, toolChoice string, messages []json.RawMessage) OpenAIRequest {
	req := OpenAIRequest{
		Model:         model,
		Messages:      prependSystemMessage(systemPrompt, messages),
		Tools:         tools,
		Stream:        true,
		StreamOptions: &StreamOptions{IncludeUsage: true},
	}
	if gptVerbosity != "" {
		req.Verbosity = &gptVerbosity
	}
	if reasoningEffort != "" {
		req.ReasoningEffort = &reasoningEffort
	}
	if toolChoice != "" && len(tools) > 0 {
		req.ToolChoice = &toolChoice
	}
	return req
}

// TODO: PromptCacheKey important when adding multi-user support

func encodeJSON(v any, pw *io.PipeWriter) {
	pw.CloseWithError(json.NewEncoder(pw).Encode(v))
}

func jsonReader(v any) io.Reader {
	pr, pw := io.Pipe()
	go encodeJSON(v, pw)
	return pr
}
