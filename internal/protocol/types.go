package protocol

import "encoding/json"

type ChatRequest struct {
	Messages       []json.RawMessage `json:"messages"`
	Tools          []json.RawMessage `json:"tools,omitempty"`
	ResponseFormat json.RawMessage   `json:"response_format,omitempty"`
}

type InputMessage struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponsesRequest struct {
	Model             string            `json:"model"`
	Input             []json.RawMessage `json:"input"`
	Tools             []json.RawMessage `json:"tools,omitempty"`
	Include           []string          `json:"include,omitempty"`
	ToolChoice        *string           `json:"tool_choice,omitempty"`
	Temperature       *float64          `json:"temperature,omitempty"`
	ServiceTier       string            `json:"service_tier,omitempty"`
	Stream            bool              `json:"stream"`
	Store             bool              `json:"store"`
	Reasoning         *ReasoningConfig  `json:"reasoning,omitempty"`
	Text              *TextConfig       `json:"text,omitempty"`
	ParallelToolCalls bool              `json:"parallel_tool_calls"`
}

type ReasoningConfig struct {
	Effort  string `json:"effort"`
	Summary string `json:"summary,omitempty"`
}

type TextConfig struct {
	Format    json.RawMessage `json:"format,omitempty"`
	Verbosity string          `json:"verbosity,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens        int `json:"cached_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
}

type OutputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

type UsageResponse struct {
	InputTokens         int                  `json:"input_tokens"`
	OutputTokens        int                  `json:"output_tokens"`
	TotalTokens         int                  `json:"total_tokens"`
	InputTokensDetails  *PromptTokensDetails `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *OutputTokensDetails `json:"output_tokens_details,omitempty"`
}
