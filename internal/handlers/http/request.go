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

type chatResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

type textFormat struct {
	Type   string          `json:"type"`
	Name   string          `json:"name,omitempty"`
	Schema json.RawMessage `json:"schema,omitempty"`
	Strict bool            `json:"strict,omitempty"`
}

type jsonSchemaInner struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict"`
}

func toTextFormat(responseFormat json.RawMessage) json.RawMessage {
	if responseFormat == nil {
		return nil
	}
	var chatFmt chatResponseFormat
	if err := json.Unmarshal(responseFormat, &chatFmt); err != nil {
		return nil
	}
	if chatFmt.Type != "json_schema" || chatFmt.JSONSchema == nil {
		return responseFormat
	}
	var inner jsonSchemaInner
	if err := json.Unmarshal(chatFmt.JSONSchema, &inner); err != nil {
		return nil
	}
	result, err := json.Marshal(textFormat{
		Type:   chatFmt.Type,
		Name:   inner.Name,
		Schema: inner.Schema,
		Strict: inner.Strict,
	})
	if err != nil {
		return nil
	}
	return result
}

func buildTextConfig(verbosity string, responseFormat json.RawMessage) *TextConfig {
	format := toTextFormat(responseFormat)
	if verbosity == "" && format == nil {
		return nil
	}
	return &TextConfig{Format: format, Verbosity: verbosity}
}

func buildResponsesRequest(model, systemPrompt, reasoningEffort, reasoningSummary, verbosity string, tools []json.RawMessage, toolChoice string, temperature *float64, messages []json.RawMessage, responseFormat json.RawMessage) ResponsesRequest {
	req := ResponsesRequest{
		Model:             model,
		Input:             prependSystemMessage(systemPrompt, messages),
		Tools:             tools,
		Include:           []string{"reasoning.encrypted_content"},
		Stream:            true,
		Store:             false,
		ParallelToolCalls: true,
		Text:              buildTextConfig(verbosity, responseFormat),
	}
	if temperature != nil {
		req.Temperature = temperature
	}
	if reasoningEffort == "off" {
		reasoningEffort = ""
	}
	if reasoningEffort != "" || reasoningSummary != "" {
		req.Reasoning = &ReasoningConfig{Effort: reasoningEffort, Summary: reasoningSummary}
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
