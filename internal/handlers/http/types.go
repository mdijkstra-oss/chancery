package http

import (
	"encoding/json"

	"github.com/sashabaranov/go-openai"
)

type ChatRequest struct {
	Messages []json.RawMessage `json:"messages"`
}

type Config struct {
	APIKey  string
	BaseURL string
	Verbose bool
	Pricing Pricing
}

type SystemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model                string            `json:"model"`
	Tools                []openai.Tool     `json:"tools,omitempty"`
	ToolChoice           *string           `json:"tool_choice,omitempty"`
	Temperature          *float64          `json:"temperature,omitempty"`
	Messages             []json.RawMessage `json:"messages"`
	Stream               bool              `json:"stream"`
	StreamOptions        *StreamOptions    `json:"stream_options,omitempty"`
	Verbosity            *string           `json:"verbosity,omitempty"`
	ReasoningEffort      *string           `json:"reasoning_effort,omitempty"`
	PromptCacheRetention *string           `json:"prompt_cache_retention,omitempty"`
	PromptCacheKey       *string           `json:"prompt_cache_key,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type StreamChunk struct {
	Usage *UsageResponse `json:"usage,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type UsageResponse struct {
	PromptTokens        int                  `json:"prompt_tokens"`
	CompletionTokens    int                  `json:"completion_tokens"`
	TotalTokens         int                  `json:"total_tokens"`
	PromptTokensDetails *PromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

type Message struct {
	Role       string        `json:"role"`
	Content    string        `json:"content,omitempty"`
	ToolCalls  []interface{} `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}
