package protocol

import (
	"encoding/json"

	"github.com/matthijn/hermes-logos/internal/fn"
)

type RequestParams struct {
	Model            string
	SystemPrompt     string
	ReasoningEffort  string
	ReasoningSummary string
	Verbosity        string
	ServiceTier      string
	ToolChoice       string
	LegacyThinking   bool
	Temperature      *float64
	Seed             bool
	MaxTokens        int
	AutoCache        bool
	CacheTTL         int
	Tools            []json.RawMessage
	Messages         []json.RawMessage
	ResponseFormat   json.RawMessage
	CacheBreakpoints map[int]bool
}

func BuildResponsesRequestFromParams(p RequestParams) ResponsesRequest {
	return buildResponsesRequest(
		p.Model, p.SystemPrompt,
		p.ReasoningEffort, p.ReasoningSummary,
		p.Verbosity, p.ServiceTier,
		p.Tools, p.ToolChoice, p.Temperature,
		StripExtraContent(p.Messages), p.ResponseFormat,
	)
}

func StripExtraContent(messages []json.RawMessage) []json.RawMessage {
	return fn.Map(messages, func(raw json.RawMessage) json.RawMessage {
		return stripField(raw, "extra_content")
	})
}

func stripField(raw json.RawMessage, field string) json.RawMessage {
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return raw
	}
	if _, ok := obj[field]; !ok {
		return raw
	}
	delete(obj, field)
	cleaned, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return cleaned
}

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

func extractJSONSchemaInner(responseFormat json.RawMessage) *jsonSchemaInner {
	var outer chatResponseFormat
	if json.Unmarshal(responseFormat, &outer) != nil || outer.Type != "json_schema" || outer.JSONSchema == nil {
		return nil
	}
	var inner jsonSchemaInner
	if json.Unmarshal(outer.JSONSchema, &inner) != nil {
		return nil
	}
	return &inner
}

func toTextFormat(responseFormat json.RawMessage) json.RawMessage {
	if responseFormat == nil {
		return nil
	}
	inner := extractJSONSchemaInner(responseFormat)
	if inner == nil {
		return responseFormat
	}
	result, err := json.Marshal(textFormat{
		Type:   "json_schema",
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

func buildResponsesRequest(model, systemPrompt, reasoningEffort, reasoningSummary, verbosity, serviceTier string, tools []json.RawMessage, toolChoice string, temperature *float64, messages []json.RawMessage, responseFormat json.RawMessage) ResponsesRequest {
	req := ResponsesRequest{
		Model:             model,
		Input:             prependSystemMessage(systemPrompt, messages),
		Tools:             tools,
		ServiceTier:       serviceTier,
		Stream:            true,
		Store:             false,
		ParallelToolCalls: true,
		Text:              buildTextConfig(verbosity, responseFormat),
	}
	if temperature != nil {
		req.Temperature = temperature
	}
	if reasoningEffort != "" && reasoningEffort != "off" {
		req.Reasoning = &ReasoningConfig{Effort: reasoningEffort, Summary: reasoningSummary}
		req.Include = []string{"reasoning.encrypted_content"}
	}
	if toolChoice != "" && len(tools) > 0 {
		req.ToolChoice = &toolChoice
	}
	return req
}
