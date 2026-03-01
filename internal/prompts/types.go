package prompts

type Segment struct {
	Source  string
	Content string
}

type Pricing struct {
	Input       float64 `json:"input"`
	Output      float64 `json:"output"`
	CachedInput float64 `json:"cached_input"`
}

type PromptConfig struct {
	Model            string  `json:"model"`
	ReasoningEffort  string  `json:"reasoning_effort"`
	ReasoningSummary string  `json:"reasoning_summary"`
	Verbosity        string  `json:"verbosity"`
	ServiceTier      string  `json:"service_tier,omitempty"`
	Pricing          Pricing `json:"pricing"`
	CompactAt        int     `json:"compact_at,omitempty"`
}
