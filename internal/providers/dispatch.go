package providers

import (
	"github.com/matthijn/hermes-logos/internal/prompts"
	"github.com/matthijn/hermes-logos/internal/providers/anthropic"
	"github.com/matthijn/hermes-logos/internal/providers/completions"
	"github.com/matthijn/hermes-logos/internal/providers/gemini"
	"github.com/matthijn/hermes-logos/internal/providers/openai"
	"github.com/matthijn/hermes-logos/internal/providers/sse"
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
