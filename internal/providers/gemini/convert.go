package gemini

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"

	"google.golang.org/genai"
	"hermes-logos/internal/protocol"
)

type messagePeek struct {
	Type             string          `json:"type"`
	Role             string          `json:"role"`
	Content          string          `json:"content"`
	Name             string          `json:"name"`
	CallID           string          `json:"call_id"`
	Arguments        string          `json:"arguments"`
	Output           string          `json:"output"`
	ID               string          `json:"id"`
	ExtraContent     json.RawMessage `json:"extra_content"`
}

type extraContent struct {
	Google *googleExtra `json:"google"`
}

type googleExtra struct {
	ThoughtSignature string `json:"thought_signature"`
}

func ExtractLeadingSystem(messages []json.RawMessage) ([]string, []json.RawMessage) {
	var leading []string
	for i, raw := range messages {
		var m messagePeek
		if json.Unmarshal(raw, &m) != nil {
			return leading, messages[i:]
		}
		if m.Role != "system" {
			return leading, messages[i:]
		}
		leading = append(leading, m.Content)
	}
	return leading, nil
}

func BuildCallIDToName(messages []json.RawMessage) map[string]string {
	result := make(map[string]string)
	for _, raw := range messages {
		var m messagePeek
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		if m.Type == "function_call" && m.CallID != "" {
			result[m.CallID] = m.Name
		}
	}
	return result
}

func MessagesToContents(messages []json.RawMessage, callIDMap map[string]string, thinkingEnabled bool) []*genai.Content {
	var contents []*genai.Content
	for _, raw := range messages {
		var m messagePeek
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		content := messageToContent(m, callIDMap, thinkingEnabled)
		if content == nil {
			continue
		}
		contents = append(contents, content)
	}
	return contents
}

func MergeConsecutiveContents(contents []*genai.Content) []*genai.Content {
	if len(contents) == 0 {
		return nil
	}
	merged := make([]*genai.Content, 0, len(contents))
	merged = append(merged, &genai.Content{
		Role:  contents[0].Role,
		Parts: append([]*genai.Part{}, contents[0].Parts...),
	})
	for _, c := range contents[1:] {
		last := merged[len(merged)-1]
		if last.Role == c.Role {
			last.Parts = append(last.Parts, c.Parts...)
		} else {
			merged = append(merged, &genai.Content{
				Role:  c.Role,
				Parts: append([]*genai.Part{}, c.Parts...),
			})
		}
	}
	return merged
}

func messageToContent(m messagePeek, callIDMap map[string]string, thinkingEnabled bool) *genai.Content {
	switch {
	case m.Type == "function_call":
		return functionCallToContent(m, thinkingEnabled)
	case m.Type == "function_call_output":
		return functionCallOutputToContent(m, callIDMap)
	case m.Type == "reasoning":
		return reasoningToContent(m)
	case m.Role == "system":
		return genai.NewContentFromText("<system_message>\n"+m.Content+"\n</system_message>", "user")
	case m.Role == "user":
		return genai.NewContentFromText(m.Content, "user")
	case m.Role == "assistant":
		return genai.NewContentFromText(m.Content, "model")
	default:
		return nil
	}
}

var fallbackThoughtSig = []byte("context_engineering_is_the_way_to_go")

func functionCallToContent(m messagePeek, thinkingEnabled bool) *genai.Content {
	var args map[string]any
	if m.Arguments != "" {
		json.Unmarshal([]byte(m.Arguments), &args)
	}
	part := genai.NewPartFromFunctionCall(m.Name, args)
	part.FunctionCall.ID = m.CallID
	sig := extractThoughtSignature(m.ExtraContent)
	if sig == nil && thinkingEnabled {
		slog.Warn("gemini function call missing thought signature, using fallback", "name", m.Name, "call_id", m.CallID)
		sig = fallbackThoughtSig
	}
	part.ThoughtSignature = sig
	return genai.NewContentFromParts([]*genai.Part{part}, "model")
}

func extractThoughtSignature(raw json.RawMessage) []byte {
	if raw == nil {
		return nil
	}
	var extra extraContent
	if json.Unmarshal(raw, &extra) != nil || extra.Google == nil || extra.Google.ThoughtSignature == "" {
		return nil
	}
	sig, err := base64.StdEncoding.DecodeString(extra.Google.ThoughtSignature)
	if err != nil {
		return nil
	}
	return sig
}

func functionCallOutputToContent(m messagePeek, callIDMap map[string]string) *genai.Content {
	name := callIDMap[m.CallID]
	var resp map[string]any
	if m.Output != "" {
		if json.Unmarshal([]byte(m.Output), &resp) != nil {
			resp = map[string]any{"output": m.Output}
		}
	}
	content := genai.NewContentFromFunctionResponse(name, resp, "user")
	content.Parts[0].FunctionResponse.ID = m.CallID
	return content
}

func reasoningToContent(m messagePeek) *genai.Content {
	sig := extractThoughtSignature(m.ExtraContent)
	if len(sig) == 0 {
		return nil
	}
	return &genai.Content{
		Role: "model",
		Parts: []*genai.Part{{
			Thought:          true,
			ThoughtSignature: sig,
		}},
	}
}

func ToolsToGemini(tools []json.RawMessage) []*genai.Tool {
	if len(tools) == 0 {
		return nil
	}
	var decls []*genai.FunctionDeclaration
	for _, raw := range tools {
		decl := toolToDeclaration(raw)
		if decl != nil {
			decls = append(decls, decl)
		}
	}
	if len(decls) == 0 {
		return nil
	}
	return []*genai.Tool{{FunctionDeclarations: decls}}
}

type toolPeek struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

func toolToDeclaration(raw json.RawMessage) *genai.FunctionDeclaration {
	var t toolPeek
	if json.Unmarshal(raw, &t) != nil || t.Name == "" {
		return nil
	}
	decl := &genai.FunctionDeclaration{
		Name:        t.Name,
		Description: t.Description,
	}
	if t.Parameters != nil {
		var schema genai.Schema
		if json.Unmarshal(t.Parameters, &schema) == nil {
			decl.Parameters = &schema
		}
	}
	return decl
}

func BuildConfig(params protocol.RequestParams, leadingSystem []string) *genai.GenerateContentConfig {
	cfg := &genai.GenerateContentConfig{}

	systemText := joinSystemText(params.SystemPrompt, leadingSystem)
	if systemText != "" {
		cfg.SystemInstruction = genai.NewContentFromText(systemText, "user")
	}

	if params.Temperature != nil {
		t := float32(*params.Temperature)
		cfg.Temperature = &t
	}

	cfg.Tools = ToolsToGemini(params.Tools)
	if len(cfg.Tools) > 0 {
		cfg.ToolConfig = buildToolConfig(params.ToolChoice)
	}

	if params.ReasoningEffort != "" && params.ReasoningEffort != "off" {
		cfg.ThinkingConfig = buildThinkingConfig(params.ReasoningEffort, params.LegacyThinking)
	}

	if params.ResponseFormat != nil {
		applyResponseFormat(cfg, params.ResponseFormat)
	}

	return cfg
}

func joinSystemText(prompt string, leading []string) string {
	if len(leading) == 0 {
		return prompt
	}
	parts := make([]string, 0, 1+len(leading))
	if prompt != "" {
		parts = append(parts, prompt)
	}
	parts = append(parts, leading...)
	return strings.Join(parts, "\n\n")
}

func buildThinkingConfig(effort string, legacy bool) *genai.ThinkingConfig {
	tc := &genai.ThinkingConfig{IncludeThoughts: true}
	if legacy {
		budget := effortToBudget(effort)
		tc.ThinkingBudget = &budget
	} else {
		tc.ThinkingLevel = effortToLevel(effort)
	}
	return tc
}

var effortLevelMap = map[string]genai.ThinkingLevel{
	"minimal": genai.ThinkingLevelMinimal,
	"low":     genai.ThinkingLevelLow,
	"medium":  genai.ThinkingLevelMedium,
	"high":    genai.ThinkingLevelHigh,
}

func effortToLevel(effort string) genai.ThinkingLevel {
	level, ok := effortLevelMap[effort]
	if !ok {
		return genai.ThinkingLevelMedium
	}
	return level
}

var effortBudgetMap = map[string]int32{
	"minimal": 1024,
	"low":     4096,
	"medium":  8192,
	"high":    16384,
}

func effortToBudget(effort string) int32 {
	budget, ok := effortBudgetMap[effort]
	if !ok {
		return 8192
	}
	return budget
}

type responseFormatPeek struct {
	Type       string          `json:"type"`
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

type jsonSchemaPeek struct {
	Schema json.RawMessage `json:"schema"`
}

var toolChoiceModeMap = map[string]genai.FunctionCallingConfigMode{
	"required": genai.FunctionCallingConfigModeAny,
	"none":     genai.FunctionCallingConfigModeNone,
}

func toolChoiceToMode(choice string) genai.FunctionCallingConfigMode {
	if mode, ok := toolChoiceModeMap[choice]; ok {
		return mode
	}
	return genai.FunctionCallingConfigModeValidated
}

func buildToolConfig(toolChoice string) *genai.ToolConfig {
	return &genai.ToolConfig{
		FunctionCallingConfig: &genai.FunctionCallingConfig{
			Mode: toolChoiceToMode(toolChoice),
		},
	}
}

func applyResponseFormat(cfg *genai.GenerateContentConfig, responseFormat json.RawMessage) {
	var peek responseFormatPeek
	if json.Unmarshal(responseFormat, &peek) != nil {
		return
	}
	if peek.Type == "json_schema" && peek.JSONSchema != nil {
		cfg.ResponseMIMEType = "application/json"
		var inner jsonSchemaPeek
		if json.Unmarshal(peek.JSONSchema, &inner) == nil && inner.Schema != nil {
			var schema genai.Schema
			if json.Unmarshal(inner.Schema, &schema) == nil {
				cfg.ResponseSchema = &schema
			}
		}
	}
}
