package prompts

type Segment struct {
	Source  string
	Content string
}

// PromptConfig is everything one route contributes to a request body: the model its
// alias names and the fields the alias and the agent frontmatter agree on.
type PromptConfig struct {
	Model            string `json:"model"`
	Prompt           string `json:"prompt,omitempty"`
	MaxTokens        int    `json:"max_tokens,omitempty"`
	ReasoningEffort  string `json:"reasoning_effort,omitempty"`
	ReasoningSummary string `json:"reasoning_summary,omitempty"`
	Verbosity        string `json:"verbosity,omitempty"`
	ServiceTier      string `json:"service_tier,omitempty"`
}

// modelsFile is models.yaml: one flat map whose every key is an alias.
type modelsFile struct {
	Models map[string]modelEntry `yaml:"models"`
}

// modelEntry is one alias. Model is the name that travels in the body, prefix
// included; the prefix belongs to whoever serves the request and is never read here.
type modelEntry struct {
	Extends          string `yaml:"extends,omitempty"`
	Model            string `yaml:"model,omitempty"`
	Prompt           string `yaml:"prompt,omitempty"`
	MaxTokens        int    `yaml:"max_tokens,omitempty"`
	ReasoningEffort  string `yaml:"reasoning_effort,omitempty"`
	ReasoningSummary string `yaml:"reasoning_summary,omitempty"`
	Verbosity        string `yaml:"verbosity,omitempty"`
	ServiceTier      string `yaml:"service_tier,omitempty"`
}

// agentEntry is agent frontmatter, where Model names an alias rather than an upstream
// model. Prompt is a pointer because `prompt:` with no value and no `prompt:` at all
// are different mistakes: the first is a warning, the second is nothing to report.
type agentEntry struct {
	Model            string  `yaml:"model,omitempty"`
	Prompt           *string `yaml:"prompt,omitempty"`
	MaxTokens        int     `yaml:"max_tokens,omitempty"`
	ReasoningEffort  string  `yaml:"reasoning_effort,omitempty"`
	ReasoningSummary string  `yaml:"reasoning_summary,omitempty"`
	Verbosity        string  `yaml:"verbosity,omitempty"`
	ServiceTier      string  `yaml:"service_tier,omitempty"`
}

type agentFrontmatter struct {
	agentEntry  `yaml:",inline"`
	Description string                `yaml:"description,omitempty"`
	Models      map[string]agentEntry `yaml:"models,omitempty"`
	Default     string                `yaml:"default,omitempty"`
}
