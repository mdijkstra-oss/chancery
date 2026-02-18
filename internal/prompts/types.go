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
	Pricing          Pricing `json:"pricing"`
}
