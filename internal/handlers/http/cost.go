package http

type Pricing struct {
	InputCentsPerMillion       float64
	OutputCentsPerMillion      float64
	CachedInputCentsPerMillion float64
}

type Cost struct {
	InputCost  float64
	OutputCost float64
	TotalCost  float64
}

func calculateTokenCost(tokens int, centsPerMillion float64) float64 {
	return (float64(tokens) / 1_000_000) * centsPerMillion
}

func calculateInputCost(promptTokens, cachedTokens int, pricing Pricing) float64 {
	uncachedTokens := promptTokens - cachedTokens
	uncachedCost := calculateTokenCost(uncachedTokens, pricing.InputCentsPerMillion)
	cachedCost := calculateTokenCost(cachedTokens, pricing.CachedInputCentsPerMillion)
	return uncachedCost + cachedCost
}

func calculateUsageCost(pricing Pricing, usage UsageResponse) Cost {
	cachedTokens := 0
	if usage.PromptTokensDetails != nil {
		cachedTokens = usage.PromptTokensDetails.CachedTokens
	}

	inputCost := calculateInputCost(usage.PromptTokens, cachedTokens, pricing)
	outputCost := calculateTokenCost(usage.CompletionTokens, pricing.OutputCentsPerMillion)
	totalCost := inputCost + outputCost

	return Cost{
		InputCost:  inputCost,
		OutputCost: outputCost,
		TotalCost:  totalCost,
	}
}
