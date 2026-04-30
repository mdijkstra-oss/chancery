package completions

import (
	"encoding/json"
	"fmt"

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
	Thinking          *ThinkingConfig      `json:"thinking,omitempty"`
	ReasoningEffort   string               `json:"reasoning_effort,omitempty"`
}

type ThinkingConfig struct {
	Type string `json:"type"`
}

type CompletionsMessage struct {
	Role             string          `json:"role"`
	Content          string          `json:"content,omitempty"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCallEntry `json:"tool_calls,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
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

func BuildRequest(params protocol.RequestParams, strict bool) (CompletionsRequest, error) {
	thinking, effort, err := deepseekThinking(params.ReasoningEffort)
	if err != nil {
		return CompletionsRequest{}, err
	}
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
		Thinking:          thinking,
		ReasoningEffort:   effort,
	}
	if params.ToolChoice != "" && len(tools) > 0 {
		req.ToolChoice = params.ToolChoice
	}
	if params.ResponseFormat != nil {
		req.ResponseFormat = toCompletionsFormat(params.ResponseFormat)
	}
	return req, nil
}

func deepseekThinking(effort string) (*ThinkingConfig, string, error) {
	switch effort {
	case "":
		return nil, "", nil
	case "off", "none":
		return &ThinkingConfig{Type: "disabled"}, "", nil
	case "high":
		return &ThinkingConfig{Type: "enabled"}, "high", nil
	case "max":
		return &ThinkingConfig{Type: "enabled"}, "max", nil
	default:
		return nil, "", fmt.Errorf("unsupported reasoning_effort for completions: %q", effort)
	}
}

var jsonObjectFormat = json.RawMessage(`{"type":"json_object"}`)

func toCompletionsFormat(responseFormat json.RawMessage) json.RawMessage {
	var peek struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(responseFormat, &peek) != nil {
		return responseFormat
	}
	if peek.Type == "json_schema" {
		return jsonObjectFormat
	}
	return responseFormat
}

type messagePeek struct {
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Content      string          `json:"content"`
	Name         string          `json:"name"`
	CallID       string          `json:"call_id"`
	Arguments    string          `json:"arguments"`
	Output       string          `json:"output"`
	ExtraContent json.RawMessage `json:"extra_content"`
}

func MessagesToCompletions(systemPrompt string, messages []json.RawMessage) []CompletionsMessage {
	var result []CompletionsMessage
	if systemPrompt != "" {
		result = append(result, CompletionsMessage{Role: "system", Content: systemPrompt})
	}
	var pendingReasoning string
	for i := 0; i < len(messages); i++ {
		var m messagePeek
		if json.Unmarshal(messages[i], &m) != nil {
			continue
		}
		switch {
		case m.Type == "reasoning":
			pendingReasoning = extractDeepSeekReasoning(m.ExtraContent)
		case m.Type == "function_call":
			calls, consumed := collectFunctionCalls(messages, i)
			i += consumed - 1
			attachOrCreateAssistant(&result, calls, pendingReasoning)
			pendingReasoning = ""
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
			msg := CompletionsMessage{Role: "assistant", Content: m.Content, ReasoningContent: pendingReasoning}
			pendingReasoning = ""
			result = append(result, msg)
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

func attachOrCreateAssistant(result *[]CompletionsMessage, calls []ToolCallEntry, reasoning string) {
	if len(*result) > 0 && (*result)[len(*result)-1].Role == "assistant" && len((*result)[len(*result)-1].ToolCalls) == 0 {
		(*result)[len(*result)-1].ToolCalls = calls
		if reasoning != "" && (*result)[len(*result)-1].ReasoningContent == "" {
			(*result)[len(*result)-1].ReasoningContent = reasoning
		}
		return
	}
	*result = append(*result, CompletionsMessage{Role: "assistant", ToolCalls: calls, ReasoningContent: reasoning})
}

type deepseekExtraContent struct {
	DeepSeek *deepseekExtra `json:"deepseek"`
}

type deepseekExtra struct {
	ReasoningContent string `json:"reasoning_content"`
}

func extractDeepSeekReasoning(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var extra deepseekExtraContent
	if json.Unmarshal(raw, &extra) != nil || extra.DeepSeek == nil {
		return ""
	}
	return extra.DeepSeek.ReasoningContent
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

func sanitizeSchema(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return raw
	}
	if isEmptyObject(obj) {
		return nil
	}
	changed := flattenCompositions(obj)
	changed = stripUnsupportedFormat(obj) || changed
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

func flattenCompositions(obj map[string]json.RawMessage) bool {
	changed := false
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		arrRaw, ok := obj[key]
		if !ok {
			continue
		}
		var arr []json.RawMessage
		if json.Unmarshal(arrRaw, &arr) != nil {
			continue
		}
		flattened, didFlatten := flattenSchemaArray(arr, key)
		if !didFlatten {
			continue
		}
		deduped := deduplicateSchemas(flattened)
		out, err := json.Marshal(deduped)
		if err != nil {
			continue
		}
		obj[key] = out
		changed = true
	}
	return changed
}

func isTypelessComposition(inner map[string]json.RawMessage, compositionKey string) bool {
	if _, hasType := inner["type"]; hasType {
		return false
	}
	if _, hasRef := inner["$ref"]; hasRef {
		return false
	}
	_, hasComposition := inner[compositionKey]
	return hasComposition
}

func flattenSchemaArray(arr []json.RawMessage, compositionKey string) ([]json.RawMessage, bool) {
	changed := false
	result := make([]json.RawMessage, 0, len(arr))
	for _, v := range arr {
		var inner map[string]json.RawMessage
		if json.Unmarshal(v, &inner) != nil {
			result = append(result, v)
			continue
		}
		if !isTypelessComposition(inner, compositionKey) {
			result = append(result, v)
			continue
		}
		nestedRaw := inner[compositionKey]
		var nested []json.RawMessage
		if json.Unmarshal(nestedRaw, &nested) != nil {
			result = append(result, v)
			continue
		}
		result = append(result, nested...)
		changed = true
	}
	return result, changed
}

func deduplicateSchemas(arr []json.RawMessage) []json.RawMessage {
	seen := make(map[string]bool)
	result := make([]json.RawMessage, 0, len(arr))
	for _, v := range arr {
		var obj any
		json.Unmarshal(v, &obj)
		norm, _ := json.Marshal(obj)
		key := string(norm)
		if !seen[key] {
			seen[key] = true
			result = append(result, json.RawMessage(norm))
		}
	}
	return result
}

func sanitizeSchemaChildren(obj map[string]json.RawMessage) bool {
	changed := false
	for _, key := range []string{"properties", "patternProperties"} {
		if propsRaw, ok := obj[key]; ok {
			sanitized, removed := sanitizeObjectMap(propsRaw)
			if sanitized != nil {
				obj[key] = sanitized
				changed = true
			}
			if len(removed) > 0 {
				changed = stripFromRequired(obj, removed) || changed
			}
		}
	}
	for _, key := range []string{"items", "additionalProperties", "not"} {
		if child, ok := obj[key]; ok {
			result := sanitizeSchema(child)
			if result == nil {
				delete(obj, key)
				changed = true
				continue
			}
			if !jsonEqual(child, result) {
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

func stripFromRequired(obj map[string]json.RawMessage, removed []string) bool {
	reqRaw, ok := obj["required"]
	if !ok {
		return false
	}
	var required []string
	if json.Unmarshal(reqRaw, &required) != nil {
		return false
	}
	removedSet := make(map[string]bool, len(removed))
	for _, r := range removed {
		removedSet[r] = true
	}
	filtered := make([]string, 0, len(required))
	for _, r := range required {
		if !removedSet[r] {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == len(required) {
		return false
	}
	if len(filtered) == 0 {
		delete(obj, "required")
	} else {
		out, _ := json.Marshal(filtered)
		obj["required"] = out
	}
	return true
}

func sanitizeObjectMap(raw json.RawMessage) (json.RawMessage, []string) {
	var props map[string]json.RawMessage
	if json.Unmarshal(raw, &props) != nil {
		return nil, nil
	}
	changed := false
	var removed []string
	for k, v := range props {
		result := sanitizeSchema(v)
		if result == nil {
			delete(props, k)
			removed = append(removed, k)
			changed = true
			continue
		}
		if !jsonEqual(v, result) {
			props[k] = result
			changed = true
		}
	}
	if !changed {
		return nil, nil
	}
	out, err := json.Marshal(props)
	if err != nil {
		return nil, nil
	}
	return out, removed
}

func sanitizeSchemaArray(raw json.RawMessage) json.RawMessage {
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) != nil {
		return nil
	}
	changed := false
	filtered := make([]json.RawMessage, 0, len(arr))
	for _, v := range arr {
		result := sanitizeSchema(v)
		if result == nil {
			changed = true
			continue
		}
		if !jsonEqual(v, result) {
			changed = true
		}
		filtered = append(filtered, result)
	}
	if !changed {
		return nil
	}
	out, err := json.Marshal(filtered)
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
