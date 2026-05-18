package providers

import (
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/providers/anthropic"
	"hermes-logos/internal/providers/completions"
	"hermes-logos/internal/providers/gemini"
	"hermes-logos/internal/providers/openai"
	"hermes-logos/internal/providers/sse"
)

func StreamForProtocol(p prompts.Protocol) sse.StreamFunc {
	switch p {
	case prompts.ProtocolResponses:
		return openai.Stream
	case prompts.ProtocolGemini:
		return gemini.Stream
	case prompts.ProtocolCompletions:
		return completions.Stream
	case prompts.ProtocolAnthropic:
		return anthropic.Stream
	default:
		panic("unknown protocol: " + string(p))
	}
}
