package http

import "hermes-logos/internal/prompts"

type Cost struct {
	InputCost  float64
	OutputCost float64
	TotalCost  float64
}

func calculateTokenCost(tokens int, centsPerMillion float64) float64 {
	return (float64(tokens) / 1_000_000) * centsPerMillion
}

func calculateInputCost(promptTokens, cachedTokens int, pricing prompts.Pricing) float64 {
	uncachedTokens := promptTokens - cachedTokens
	uncachedCost := calculateTokenCost(uncachedTokens, pricing.Input)
	cachedCost := calculateTokenCost(cachedTokens, pricing.CachedInput)
	return uncachedCost + cachedCost
}

func calculateUsageCost(usage UsageResponse, pricing prompts.Pricing) Cost {
	cachedTokens := 0
	if usage.InputTokensDetails != nil {
		cachedTokens = usage.InputTokensDetails.CachedTokens
	}

	inputCost := calculateInputCost(usage.InputTokens, cachedTokens, pricing)
	outputCost := calculateTokenCost(usage.OutputTokens, pricing.Output)

	return Cost{
		InputCost:  inputCost,
		OutputCost: outputCost,
		TotalCost:  inputCost + outputCost,
	}
}
