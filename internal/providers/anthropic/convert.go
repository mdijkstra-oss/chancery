package anthropic

import (
	"encoding/json"

	"hermes-logos/internal/protocol"
	"hermes-logos/internal/prompts"
)

type Request struct {
	Model       string            `json:"model"`
	MaxTokens   int               `json:"max_tokens"`
	System      []SystemBlock     `json:"system,omitempty"`
	Messages    []Message         `json:"messages"`
	Tools       []Tool            `json:"tools,omitempty"`
	ToolChoice  any               `json:"tool_choice,omitempty"`
	Thinking     *ThinkingConfig   `json:"thinking,omitempty"`
	OutputConfig *OutputConfig     `json:"output_config,omitempty"`
	Temperature  *float64          `json:"temperature,omitempty"`
	Stream      bool              `json:"stream"`
}

type SystemBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type CacheControl struct {
	Type string `json:"type"`
}

type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

type ContentBlock struct {
	Type         string          `json:"type"`
	Text         string          `json:"text,omitempty"`
	ID           string          `json:"id,omitempty"`
	Name         string          `json:"name,omitempty"`
	Input        json.RawMessage `json:"input,omitempty"`
	ToolUseID    string          `json:"tool_use_id,omitempty"`
	Content      string          `json:"content,omitempty"`
	Thinking     string          `json:"thinking,omitempty"`
	Signature    string          `json:"signature,omitempty"`
	CacheControl *CacheControl   `json:"cache_control,omitempty"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type ThinkingConfig struct {
	Type string `json:"type"`
}

type OutputConfig struct {
	Effort string `json:"effort"`
}

type messagePeek struct {
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Content      string          `json:"content"`
	Name         string          `json:"name"`
	CallID       string          `json:"call_id"`
	Arguments    string          `json:"arguments"`
	Output       string          `json:"output"`
	ID           string          `json:"id"`
	ExtraContent json.RawMessage `json:"extra_content"`
}

type anthropicExtraContent struct {
	Anthropic *anthropicExtra `json:"anthropic"`
}

type anthropicExtra struct {
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

type toolPeek struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

func BuildRequest(params protocol.RequestParams, provider prompts.ProviderConfig) Request {
	hasBreakpoints := params.CacheBreakpoints != nil

	system := SystemToAnthropic(params.SystemPrompt, hasBreakpoints)
	messages := MessagesToAnthropic(params.Messages, params.CacheBreakpoints)
	tools := ToolsToAnthropic(params.Tools)
	thinking := buildThinkingConfig(params.ReasoningEffort)
	outputConfig := buildOutputConfig(params.ReasoningEffort)
	toolChoice := buildToolChoice(params.ToolChoice)

	maxTokens := params.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	req := Request{
		Model:        params.Model,
		MaxTokens:    maxTokens,
		System:       system,
		Messages:     messages,
		Tools:        tools,
		Thinking:     thinking,
		OutputConfig: outputConfig,
		Stream:       true,
	}

	if toolChoice != nil {
		req.ToolChoice = toolChoice
	}

	isThinkingEnabled := thinking != nil && thinking.Type != "disabled"
	if params.Temperature != nil && !isThinkingEnabled {
		req.Temperature = params.Temperature
	}

	return req
}

func SystemToAnthropic(systemPrompt string, hasBreakpoints bool) []SystemBlock {
	if systemPrompt == "" {
		return nil
	}
	block := SystemBlock{
		Type: "text",
		Text: systemPrompt,
	}
	if hasBreakpoints {
		block.CacheControl = &CacheControl{Type: "ephemeral"}
	}
	return []SystemBlock{block}
}

func MessagesToAnthropic(messages []json.RawMessage, breakpoints map[int]bool) []Message {
	var result []Message
	for i, raw := range messages {
		var m messagePeek
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		msg := messageToAnthropic(m)
		if msg == nil {
			continue
		}
		if breakpoints[i] && len(msg.Content) > 0 {
			last := &msg.Content[len(msg.Content)-1]
			last.CacheControl = &CacheControl{Type: "ephemeral"}
		}
		result = mergeConsecutive(result, *msg)
	}
	return result
}

func messageToAnthropic(m messagePeek) *Message {
	switch {
	case m.Type == "function_call":
		return functionCallToAnthropic(m)
	case m.Type == "function_call_output":
		return functionCallOutputToAnthropic(m)
	case m.Type == "reasoning":
		return reasoningToAnthropic(m)
	case m.Role == "system":
		return &Message{
			Role: "user",
			Content: []ContentBlock{{
				Type: "text",
				Text: "<system_message>\n" + m.Content + "\n</system_message>",
			}},
		}
	case m.Role == "user":
		return &Message{
			Role:    "user",
			Content: []ContentBlock{{Type: "text", Text: m.Content}},
		}
	case m.Role == "assistant":
		return &Message{
			Role:    "assistant",
			Content: []ContentBlock{{Type: "text", Text: m.Content}},
		}
	default:
		return nil
	}
}

func functionCallToAnthropic(m messagePeek) *Message {
	var input json.RawMessage
	if m.Arguments != "" {
		input = json.RawMessage(m.Arguments)
	} else {
		input = json.RawMessage(`{}`)
	}
	blocks := []ContentBlock{{
		Type:  "tool_use",
		ID:    m.CallID,
		Name:  m.Name,
		Input: input,
	}}
	thinking, signature := extractAnthropicThinking(m.ExtraContent)
	if thinking != "" || signature != "" {
		blocks = append([]ContentBlock{{
			Type:      "thinking",
			Thinking:  thinking,
			Signature: signature,
		}}, blocks...)
	}
	return &Message{Role: "assistant", Content: blocks}
}

func functionCallOutputToAnthropic(m messagePeek) *Message {
	return &Message{
		Role: "user",
		Content: []ContentBlock{{
			Type:      "tool_result",
			ToolUseID: m.CallID,
			Content:   m.Output,
		}},
	}
}

func reasoningToAnthropic(m messagePeek) *Message {
	thinking, signature := extractAnthropicThinking(m.ExtraContent)
	if thinking == "" && signature == "" {
		return nil
	}
	return &Message{
		Role: "assistant",
		Content: []ContentBlock{{
			Type:      "thinking",
			Thinking:  thinking,
			Signature: signature,
		}},
	}
}

func extractAnthropicThinking(raw json.RawMessage) (string, string) {
	if raw == nil {
		return "", ""
	}
	var extra anthropicExtraContent
	if json.Unmarshal(raw, &extra) != nil || extra.Anthropic == nil {
		return "", ""
	}
	return extra.Anthropic.Thinking, extra.Anthropic.Signature
}

func mergeConsecutive(messages []Message, msg Message) []Message {
	if len(messages) == 0 {
		return append(messages, msg)
	}
	last := &messages[len(messages)-1]
	if last.Role == msg.Role {
		last.Content = append(last.Content, msg.Content...)
		return messages
	}
	return append(messages, msg)
}

func ToolsToAnthropic(tools []json.RawMessage) []Tool {
	if len(tools) == 0 {
		return nil
	}
	var result []Tool
	for _, raw := range tools {
		var t toolPeek
		if json.Unmarshal(raw, &t) != nil || t.Name == "" {
			continue
		}
		result = append(result, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func buildThinkingConfig(effort string) *ThinkingConfig {
	switch effort {
	case "", "off", "none":
		return &ThinkingConfig{Type: "disabled"}
	default:
		return &ThinkingConfig{Type: "adaptive"}
	}
}

var effortMap = map[string]string{
	"minimal": "low",
	"low":     "low",
	"medium":  "medium",
	"high":    "high",
	"max":     "max",
}

func buildOutputConfig(effort string) *OutputConfig {
	mapped, ok := effortMap[effort]
	if !ok {
		return nil
	}
	return &OutputConfig{Effort: mapped}
}

func buildToolChoice(choice string) any {
	switch choice {
	case "":
		return nil
	case "required":
		return map[string]string{"type": "any"}
	case "none":
		return map[string]string{"type": "none"}
	default:
		return map[string]string{"type": "tool", "name": choice}
	}
}
