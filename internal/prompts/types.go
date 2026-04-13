package prompts

type Protocol string

const (
	ProtocolResponses Protocol = "responses"
	ProtocolChat      Protocol = "chat"
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
}

type ProviderConfig struct {
	Protocol Protocol
	BaseURL  string
	APIKey   string
}

type PromptConfig struct {
	Model            string         `json:"model"`
	Dimensions       int            `json:"dimensions,omitempty"`
	ReasoningEffort  string         `json:"reasoning_effort"`
	ReasoningSummary string         `json:"reasoning_summary"`
	Verbosity        string         `json:"verbosity"`
	ServiceTier      string         `json:"service_tier,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	Pricing          Pricing        `json:"pricing"`
	CompactAt        int            `json:"compact_at,omitempty"`
	Provider         ProviderConfig `json:"provider"`
}

type agentsFile struct {
	Providers map[string]ProviderEntry `json:"providers"`
	Models    map[string]modelEntry    `json:"models"`
	Agents    map[string]agentEntry    `json:"agents"`
}

type modelEntry struct {
	Extends          string  `json:"extends,omitempty"`
	Provider         string  `json:"provider,omitempty"`
	Name             string  `json:"name,omitempty"`
	Type             string  `json:"type,omitempty"`
	Dimensions       int     `json:"dimensions,omitempty"`
	ReasoningEffort  string  `json:"reasoning_effort,omitempty"`
	ReasoningSummary string  `json:"reasoning_summary,omitempty"`
	Verbosity        string  `json:"verbosity,omitempty"`
	ServiceTier      string  `json:"service_tier,omitempty"`
	CompactAt        int     `json:"compact_at,omitempty"`
	Pricing          Pricing `json:"pricing"`
}

type agentEntry struct {
	Model            string   `json:"model"`
	ReasoningEffort  string   `json:"reasoning_effort,omitempty"`
	ReasoningSummary string   `json:"reasoning_summary,omitempty"`
	Verbosity        string   `json:"verbosity,omitempty"`
	ServiceTier      string   `json:"service_tier,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	CompactAt        int      `json:"compact_at,omitempty"`
	Dimensions       int      `json:"dimensions,omitempty"`
}
