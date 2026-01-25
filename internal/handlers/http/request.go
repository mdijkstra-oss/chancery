package http

import (
	"encoding/json"
	"io"
)

func prependSystemMessage(systemPrompt string, messages []json.RawMessage) []json.RawMessage {
	sysMsg := InputMessage{Type: "message", Role: "system", Content: systemPrompt}
	sysMsgJSON, _ := json.Marshal(sysMsg)
	return append([]json.RawMessage{sysMsgJSON}, messages...)
}

func buildResponsesRequest(model, systemPrompt, reasoningEffort, verbosity string, tools []json.RawMessage, toolChoice string, temperature *float64, messages []json.RawMessage) ResponsesRequest {
	req := ResponsesRequest{
		Model:             model,
		Input:             prependSystemMessage(systemPrompt, messages),
		Tools:             tools,
		Stream:            true,
		Store:             false,
		ParallelToolCalls: true,
	}
	if temperature != nil {
		req.Temperature = temperature
	}
	if reasoningEffort != "" {
		req.Reasoning = &ReasoningConfig{Effort: reasoningEffort}
	}
	if verbosity != "" {
		req.Text = &TextConfig{Verbosity: verbosity}
	}
	if toolChoice != "" && len(tools) > 0 {
		req.ToolChoice = &toolChoice
	}
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
