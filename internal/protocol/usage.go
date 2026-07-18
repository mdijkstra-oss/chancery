package protocol

func CachedInputTokens(usage *UsageResponse) int {
	if usage == nil || usage.InputTokensDetails == nil {
		return 0
	}
	return usage.InputTokensDetails.CachedTokens
}

func CacheWriteTokens(usage *UsageResponse) int {
	if usage == nil || usage.InputTokensDetails == nil {
		return 0
	}
	return usage.InputTokensDetails.CacheCreationTokens
}

func ReasoningTokens(usage *UsageResponse) int {
	if usage == nil || usage.OutputTokensDetails == nil {
		return 0
	}
	return usage.OutputTokensDetails.ReasoningTokens
}
