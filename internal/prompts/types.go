package prompts

type Protocol string

const (
	ProtocolResponses   Protocol = "responses"
	ProtocolGemini      Protocol = "gemini"
	ProtocolCompletions Protocol = "completions"
	ProtocolAnthropic   Protocol = "anthropic"
)

type Segment struct {
	Source  string
	Content string
}

type ProviderEntry struct {
	Protocol  Protocol `json:"protocol" yaml:"protocol"`
	BaseURL   string   `json:"base_url" yaml:"base_url"`
	APIKeyEnv string   `json:"api_key_env" yaml:"api_key_env"`
	Strict    bool     `json:"strict,omitempty" yaml:"strict,omitempty"`
}

type ProviderConfig struct {
	Key       string
	Protocol  Protocol
	BaseURL   string
	APIKeyEnv string
	APIKey    string
	Strict    bool
}

type PromptConfig struct {
	Model            string         `json:"model"`
	Prompt           string         `json:"prompt,omitempty"`
	Dimensions       int            `json:"dimensions,omitempty"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	ReasoningEffort  string         `json:"reasoning_effort"`
	ReasoningSummary string         `json:"reasoning_summary"`
	Verbosity        string         `json:"verbosity"`
	ServiceTier      string         `json:"service_tier,omitempty"`
	LegacyThinking   bool           `json:"legacy_thinking,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	Seed             bool           `json:"seed,omitempty"`
	AutoCache        bool           `json:"auto_cache,omitempty"`
	CacheTTL         int            `json:"cache_ttl,omitempty"`
	Provider         ProviderConfig `json:"provider"`
}

type providerFile struct {
	Protocol  Protocol              `yaml:"protocol"`
	BaseURL   string                `yaml:"base_url"`
	APIKeyEnv string                `yaml:"api_key_env"`
	Strict    bool                  `yaml:"strict,omitempty"`
	Models    map[string]modelEntry `yaml:"models"`
}

type providersFile struct {
	Providers map[string]providerFile `yaml:"providers"`
}

type modelEntry struct {
	Extends          string `yaml:"extends,omitempty"`
	Provider         string `yaml:"provider,omitempty"`
	Name             string `yaml:"name,omitempty"`
	Prompt           string `yaml:"prompt,omitempty"`
	Type             string `yaml:"type,omitempty"`
	Dimensions       int    `yaml:"dimensions,omitempty"`
	MaxTokens        int    `yaml:"max_tokens,omitempty"`
	ReasoningEffort  string `yaml:"reasoning_effort,omitempty"`
	ReasoningSummary string `yaml:"reasoning_summary,omitempty"`
	Verbosity        string `yaml:"verbosity,omitempty"`
	ServiceTier      string `yaml:"service_tier,omitempty"`
	LegacyThinking   bool   `yaml:"legacy_thinking,omitempty"`
	AutoCache        bool   `yaml:"auto_cache,omitempty"`
	CacheTTL         int    `yaml:"cache_ttl,omitempty"`
}

type agentEntry struct {
	Model            string   `yaml:"model,omitempty"`
	Prompt           *string  `yaml:"prompt,omitempty"`
	ReasoningEffort  string   `yaml:"reasoning_effort,omitempty"`
	ReasoningSummary string   `yaml:"reasoning_summary,omitempty"`
	Verbosity        string   `yaml:"verbosity,omitempty"`
	ServiceTier      string   `yaml:"service_tier,omitempty"`
	LegacyThinking   *bool    `yaml:"legacy_thinking,omitempty"`
	Temperature      *float64 `yaml:"temperature,omitempty"`
	Seed             *bool    `yaml:"seed,omitempty"`
	AutoCache        *bool    `yaml:"auto_cache,omitempty"`
	CacheTTL         int      `yaml:"cache_ttl,omitempty"`
	Dimensions       int      `yaml:"dimensions,omitempty"`
	MaxTokens        int      `yaml:"max_tokens,omitempty"`
}

type agentFrontmatter struct {
	agentEntry  `yaml:",inline"`
	Description string                `yaml:"description,omitempty"`
	Models      map[string]agentEntry `yaml:"models,omitempty"`
	Default     string                `yaml:"default,omitempty"`
}
