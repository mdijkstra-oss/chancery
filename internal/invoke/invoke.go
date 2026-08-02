package invoke

import (
	"context"
	"io"

	"github.com/mdijkstra-oss/chancery/internal/pipeline"
	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/protocol"
	"github.com/mdijkstra-oss/chancery/internal/providers"
	"github.com/mdijkstra-oss/chancery/internal/providers/openai"
	"github.com/mdijkstra-oss/chancery/internal/providers/sse"
)

type Kind string

type Target struct {
	Kind   Kind
	Agent  prompts.ResolvedAgent
	Config prompts.PromptConfig
}

const (
	KindChat       Kind = "chat"
	KindEmbeddings Kind = "embeddings"
)

func Resolve(reference string, registry prompts.Registry, lookupEnv func(string) string) (Target, error) {
	resolved, err := registry.ResolveAgent(reference)
	if err != nil {
		return Target{}, err
	}
	cfg, err := prompts.ResolveAPIKey(resolved.Config, lookupEnv)
	if err != nil {
		return Target{}, err
	}
	kind := KindChat
	if resolved.Path == "embeddings" {
		kind = KindEmbeddings
	}
	return Target{Kind: kind, Agent: resolved, Config: cfg}, nil
}

func Chat(ctx context.Context, target Target, request protocol.ChatRequest, registry prompts.Registry, output io.Writer) (sse.StreamResult, error) {
	params, _, err := pipeline.BuildRequestParamsForAgent(target.Agent, request, registry)
	if err != nil {
		return sse.StreamResult{}, err
	}
	stream := providers.StreamForProtocol(target.Config.Provider.Protocol)
	return stream(ctx, output, params, target.Config.Provider)
}

func Embeddings(ctx context.Context, target Target, inputs []string) (openai.EmbedResult, error) {
	return openai.Embed(ctx, inputs, target.Config.Model, target.Config.Dimensions, target.Config.Provider)
}
