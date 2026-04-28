package prompts

type Protocol string

const (
	ProtocolResponses   Protocol = "responses"
	ProtocolGemini      Protocol = "gemini"
	ProtocolCompletions Protocol = "completions"
)

type Segment struct {
	Source  string
	Content string
}

type Pricing struct {
	Input       float64 `json:"input"`
	Output      float64 `json:"output"`
	CachedInput float64 `json:"cached_input"`
}

type ProviderEntry struct {
	Protocol  Protocol `json:"protocol"`
	BaseURL   string   `json:"base_url"`
	APIKeyEnv string   `json:"api_key_env"`
	Strict    bool     `json:"strict,omitempty"`
}

type ProviderConfig struct {
	Key      string
	Protocol Protocol
	BaseURL  string
	APIKey   string
	Strict   bool
}

type PromptConfig struct {
	Model            string         `json:"model"`
	Prompt           string         `json:"prompt,omitempty"`
	Dimensions       int            `json:"dimensions,omitempty"`
	ReasoningEffort  string         `json:"reasoning_effort"`
	ReasoningSummary string         `json:"reasoning_summary"`
	Verbosity        string         `json:"verbosity"`
	ServiceTier      string         `json:"service_tier,omitempty"`
	LegacyThinking  bool           `json:"legacy_thinking,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	Seed             bool           `json:"seed,omitempty"`
	Pricing          Pricing        `json:"pricing"`
	CompactAt        int            `json:"compact_at,omitempty"`
	Provider         ProviderConfig `json:"provider"`
}

type providerFile struct {
	Protocol  Protocol              `json:"protocol"`
	BaseURL   string                `json:"base_url"`
	APIKeyEnv string                `json:"api_key_env"`
	Strict    bool                  `json:"strict,omitempty"`
	Models    map[string]modelEntry `json:"models"`
}

type modelEntry struct {
	Extends          string  `json:"extends,omitempty"`
	Provider         string  `json:"provider,omitempty"`
	Name             string  `json:"name,omitempty"`
	Prompt           string  `json:"prompt,omitempty"`
	Type             string  `json:"type,omitempty"`
	Dimensions       int     `json:"dimensions,omitempty"`
	ReasoningEffort  string  `json:"reasoning_effort,omitempty"`
	ReasoningSummary string  `json:"reasoning_summary,omitempty"`
	Verbosity        string  `json:"verbosity,omitempty"`
	ServiceTier      string  `json:"service_tier,omitempty"`
	LegacyThinking  bool    `json:"legacy_thinking,omitempty"`
	CompactAt        int     `json:"compact_at,omitempty"`
	Pricing          Pricing `json:"pricing"`
}

type agentEntry struct {
	Model            string   `json:"model"`
	Prompt           *string  `json:"prompt,omitempty"`
	ReasoningEffort  string   `json:"reasoning_effort,omitempty"`
	ReasoningSummary string   `json:"reasoning_summary,omitempty"`
	Verbosity        string   `json:"verbosity,omitempty"`
	ServiceTier      string   `json:"service_tier,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	Seed             bool     `json:"seed,omitempty"`
	CompactAt        int      `json:"compact_at,omitempty"`
	Dimensions       int      `json:"dimensions,omitempty"`
}
