package completions

import (
	"encoding/json"

	"hermes-logos/internal/protocol"
)

type CompletionsRequest struct {
	Model             string               `json:"model"`
	Messages          []CompletionsMessage `json:"messages"`
	Tools             []CompletionsTool    `json:"tools,omitempty"`
	Stream            bool                 `json:"stream"`
	StreamOptions     *StreamOptions       `json:"stream_options,omitempty"`
	Temperature       *float64             `json:"temperature,omitempty"`
	ToolChoice        string               `json:"tool_choice,omitempty"`
	ResponseFormat    json.RawMessage      `json:"response_format,omitempty"`
	ParallelToolCalls *bool                `json:"parallel_tool_calls,omitempty"`
}

type CompletionsMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  []ToolCallEntry `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type ToolCallEntry struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type CompletionsTool struct {
	Type     string             `json:"type"`
	Function CompletionsToolDef `json:"function"`
}

type CompletionsToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

func BuildRequest(params protocol.RequestParams, strict bool) CompletionsRequest {
	messages := MessagesToCompletions(params.SystemPrompt, params.Messages)
	tools := ToolsToCompletions(params.Tools, strict)
	parallel := true
	req := CompletionsRequest{
		Model:             params.Model,
		Messages:          messages,
		Tools:             tools,
		Stream:            true,
		StreamOptions:     &StreamOptions{IncludeUsage: true},
		Temperature:       params.Temperature,
		ParallelToolCalls: &parallel,
	}
	if params.ToolChoice != "" && len(tools) > 0 {
		req.ToolChoice = params.ToolChoice
	}
	if params.ResponseFormat != nil {
		req.ResponseFormat = params.ResponseFormat
	}
	return req
}

type messagePeek struct {
	Type      string `json:"type"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Name      string `json:"name"`
	CallID    string `json:"call_id"`
	Arguments string `json:"arguments"`
	Output    string `json:"output"`
}

func MessagesToCompletions(systemPrompt string, messages []json.RawMessage) []CompletionsMessage {
	var result []CompletionsMessage
	if systemPrompt != "" {
		result = append(result, CompletionsMessage{Role: "system", Content: systemPrompt})
	}
	for i := 0; i < len(messages); i++ {
		var m messagePeek
		if json.Unmarshal(messages[i], &m) != nil {
			continue
		}
		switch {
		case m.Type == "reasoning":
			continue
		case m.Type == "function_call":
			calls, consumed := collectFunctionCalls(messages, i)
			i += consumed - 1
			attachOrCreateAssistant(&result, calls)
		case m.Type == "function_call_output":
			result = append(result, CompletionsMessage{
				Role:       "tool",
				Content:    m.Output,
				ToolCallID: m.CallID,
			})
		case m.Role == "system":
			result = append(result, CompletionsMessage{Role: "system", Content: m.Content})
		case m.Role == "user":
			result = append(result, CompletionsMessage{Role: "user", Content: m.Content})
		case m.Role == "assistant":
			result = append(result, CompletionsMessage{Role: "assistant", Content: m.Content})
		}
	}
	return result
}

func collectFunctionCalls(messages []json.RawMessage, start int) ([]ToolCallEntry, int) {
	var calls []ToolCallEntry
	i := start
	for i < len(messages) {
		var m messagePeek
		if json.Unmarshal(messages[i], &m) != nil || m.Type != "function_call" {
			break
		}
		calls = append(calls, ToolCallEntry{
			ID:   m.CallID,
			Type: "function",
			Function: ToolCallFunction{
				Name:      m.Name,
				Arguments: m.Arguments,
			},
		})
		i++
	}
	return calls, i - start
}

func attachOrCreateAssistant(result *[]CompletionsMessage, calls []ToolCallEntry) {
	if len(*result) > 0 && (*result)[len(*result)-1].Role == "assistant" && len((*result)[len(*result)-1].ToolCalls) == 0 {
		(*result)[len(*result)-1].ToolCalls = calls
		return
	}
	*result = append(*result, CompletionsMessage{Role: "assistant", ToolCalls: calls})
}

type toolPeek struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

func ToolsToCompletions(tools []json.RawMessage, strict bool) []CompletionsTool {
	if len(tools) == 0 {
		return nil
	}
	result := make([]CompletionsTool, 0, len(tools))
	for _, raw := range tools {
		var t toolPeek
		if json.Unmarshal(raw, &t) != nil || t.Name == "" {
			continue
		}
		def := CompletionsToolDef{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  sanitizeSchema(t.Parameters),
		}
		if strict {
			s := true
			def.Strict = &s
		}
		result = append(result, CompletionsTool{Type: "function", Function: def})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

var supportedFormats = map[string]bool{
	"email":    true,
	"hostname": true,
	"ipv4":     true,
	"ipv6":     true,
	"uuid":     true,
}

var emptySchemaJSON = json.RawMessage("{}")

func sanitizeSchema(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return raw
	}
	if isEmptyObject(obj) {
		return emptySchemaJSON
	}
	changed := stripUnsupportedFormat(obj)
	changed = stripUnsupportedKeywords(obj) || changed
	changed = sanitizeSchemaChildren(obj) || changed
	if !changed {
		return raw
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return out
}

func isEmptyObject(obj map[string]json.RawMessage) bool {
	return isObjectType(obj) && !hasNonEmptyProperties(obj)
}

func isObjectType(obj map[string]json.RawMessage) bool {
	typeRaw, ok := obj["type"]
	if !ok {
		return false
	}
	var t string
	return json.Unmarshal(typeRaw, &t) == nil && t == "object"
}

func hasNonEmptyProperties(obj map[string]json.RawMessage) bool {
	propsRaw, ok := obj["properties"]
	if !ok {
		return false
	}
	var props map[string]json.RawMessage
	if json.Unmarshal(propsRaw, &props) != nil {
		return false
	}
	return len(props) > 0
}

func stripUnsupportedFormat(obj map[string]json.RawMessage) bool {
	formatRaw, ok := obj["format"]
	if !ok {
		return false
	}
	var format string
	if json.Unmarshal(formatRaw, &format) != nil {
		return false
	}
	if supportedFormats[format] {
		return false
	}
	delete(obj, "format")
	return true
}

var unsupportedStringKeywords = []string{"minLength", "maxLength"}
var unsupportedArrayKeywords = []string{"minItems", "maxItems"}

func stripUnsupportedKeywords(obj map[string]json.RawMessage) bool {
	typeRaw, ok := obj["type"]
	if !ok {
		return false
	}
	var t string
	if json.Unmarshal(typeRaw, &t) != nil {
		return false
	}
	var keywords []string
	switch t {
	case "string":
		keywords = unsupportedStringKeywords
	case "array":
		keywords = unsupportedArrayKeywords
	default:
		return false
	}
	changed := false
	for _, key := range keywords {
		if _, ok := obj[key]; ok {
			delete(obj, key)
			changed = true
		}
	}
	return changed
}

func sanitizeSchemaChildren(obj map[string]json.RawMessage) bool {
	changed := false
	for _, key := range []string{"properties", "patternProperties"} {
		if propsRaw, ok := obj[key]; ok {
			if sanitized := sanitizeObjectMap(propsRaw); sanitized != nil {
				obj[key] = sanitized
				changed = true
			}
		}
	}
	for _, key := range []string{"items", "additionalProperties", "not"} {
		if child, ok := obj[key]; ok {
			if result := sanitizeSchema(child); !jsonEqual(child, result) {
				obj[key] = result
				changed = true
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if arrRaw, ok := obj[key]; ok {
			if sanitized := sanitizeSchemaArray(arrRaw); sanitized != nil {
				obj[key] = sanitized
				changed = true
			}
		}
	}
	return changed
}

func sanitizeObjectMap(raw json.RawMessage) json.RawMessage {
	var props map[string]json.RawMessage
	if json.Unmarshal(raw, &props) != nil {
		return nil
	}
	changed := false
	for k, v := range props {
		result := sanitizeSchema(v)
		if !jsonEqual(v, result) {
			props[k] = result
			changed = true
		}
	}
	if !changed {
		return nil
	}
	out, err := json.Marshal(props)
	if err != nil {
		return nil
	}
	return out
}

func sanitizeSchemaArray(raw json.RawMessage) json.RawMessage {
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) != nil {
		return nil
	}
	changed := false
	for i, v := range arr {
		result := sanitizeSchema(v)
		if !jsonEqual(v, result) {
			arr[i] = result
			changed = true
		}
	}
	if !changed {
		return nil
	}
	out, err := json.Marshal(arr)
	if err != nil {
		return nil
	}
	return out
}

func jsonEqual(a, b json.RawMessage) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
