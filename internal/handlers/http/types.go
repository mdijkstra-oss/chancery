package http

import (
	"encoding/json"

	"github.com/sashabaranov/go-openai"
)

type ChatRequest struct {
	Messages []json.RawMessage `json:"messages"`
}

type Config struct {
	APIKey           string
	BaseURL          string
	Model            string
	Provider         string
	SystemPrompt     string
	Tools            []openai.Tool
	Verbose          bool
	IncludeReasoning bool
	CacheInterval    int
	MaxTokenWindow   int
}

type CacheControl struct {
	Type string `json:"type"`
}

type SystemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolWithCache struct {
	openai.Tool
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type OpenAIRequest struct {
	Model            string            `json:"model"`
	Messages         []json.RawMessage `json:"messages"`
	Tools            []ToolWithCache   `json:"tools,omitempty"`
	Stream           bool              `json:"stream"`
	Usage            *UsageRequest     `json:"usage,omitempty"`
	Provider         *ProviderPreference `json:"provider,omitempty"`
	IncludeReasoning *bool             `json:"include_reasoning,omitempty"`
}

type UsageRequest struct {
	Include bool `json:"include"`
}

type ProviderPreference struct {
	Only []string `json:"only"`
}

type MessageWithCache struct {
	Role         string        `json:"role"`
	Content      string        `json:"content,omitempty"`
	ToolCalls    interface{}   `json:"tool_calls,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type CacheBreakpointInfo struct {
	MessageIndex  int
	TokenPos      int
	BreakpointNum int
}

type StreamChunk struct {
	Usage *UsageResponse `json:"usage,omitempty"`
}

type UsageResponse struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CacheDiscount    float64 `json:"cache_discount,omitempty"`
}

type TokenBreakdown struct {
	System         int
	ToolDefs       int
	UserMsgs       int
	AssistantMsgs  int
	ToolCalls      int
	ToolResponses  int
}

type Message struct {
	Role       string        `json:"role"`
	Content    string        `json:"content,omitempty"`
	ToolCalls  []interface{} `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}
