package providers

import (
	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/providers/anthropic"
	"github.com/mdijkstra-oss/chancery/internal/providers/completions"
	"github.com/mdijkstra-oss/chancery/internal/providers/gemini"
	"github.com/mdijkstra-oss/chancery/internal/providers/openai"
	"github.com/mdijkstra-oss/chancery/internal/providers/sse"
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
