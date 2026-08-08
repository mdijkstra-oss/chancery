package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mdijkstra-oss/chancery/internal/auth"
	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/ratelimit"
	"github.com/mdijkstra-oss/chancery/internal/responses"

	"github.com/go-chi/chi/v5"
)

func NewChatHandler(
	registry prompts.Registry,
	client *responses.Client,
	limiter *ratelimit.Limiter,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, registry, client, limiter)
	}
}

func handleChat(
	w http.ResponseWriter,
	r *http.Request,
	registry prompts.Registry,
	client *responses.Client,
	limiter *ratelimit.Limiter,
) {
	urlPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	resolved, err := registry.ResolveAgent(urlPath)
	if err != nil {
		// A stock OpenAI SDK appends /responses to its base_url. The suffix is
		// SDK plumbing, not part of the agent's identity, so the stripped path is
		// what resolves, gets logged and travels as X-Agent — but only after the
		// exact path missed, so an agent literally named responses keeps its own.
		if stripped, ok := strings.CutSuffix(urlPath, "/responses"); ok {
			urlPath = stripped
			resolved, err = registry.ResolveAgent(urlPath)
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agent := responses.AgentFrom(resolved)
	toolPrompt, _, err := prompts.LoadToolPrompts(registry.Root, responses.ToolNames(body))
	if err != nil {
		logChatError(r, "tool prompt error", urlPath, agent.Model, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	agent.Instructions = appendPrompt(agent.Instructions, toolPrompt)
	composed, err := responses.Compose(body, agent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	slog.InfoContext(r.Context(), "request forwarded",
		"component", "chat",
		slog.Group("data",
			slog.String("endpoint", urlPath),
			slog.String("model", agent.Model),
		),
	)

	request := responses.Request{
		Body:                   composed,
		Identity:               identityFor(r, urlPath),
		PromptCacheBreakpoints: resolved.Config.PromptCacheBreakpoints,
	}
	result, err := ratelimit.Do(r.Context(), limiter, agent.Model, 3,
		func(ctx context.Context) (*responses.Response, error) {
			return client.Send(ctx, request)
		})
	if err != nil {
		// The failure names the backend's URL, its address, and whatever text it
		// answered with. That belongs in the log; the caller gets the status.
		logChatError(r, "backend error", urlPath, agent.Model, err)
		status := responses.StatusFor(err)
		http.Error(w, http.StatusText(status), status)
		return
	}
	defer result.Body.Close()

	if err := responses.Relay(w, result); err != nil {
		logChatError(r, "relay error", urlPath, agent.Model, err)
	}
}

// A tool's prompt goes behind the agent's, because the agent says what it is and the
// tool says how one thing it was handed works.
func appendPrompt(instructions, addition string) string {
	if addition == "" {
		return instructions
	}
	if instructions == "" {
		return addition
	}
	return instructions + "\n\n" + addition
}

func identityFor(r *http.Request, urlPath string) responses.Identity {
	return responses.Identity{
		RequestID: RequestIDFromContext(r.Context()),
		Agent:     urlPath,
		Subject:   auth.UserFromContext(r.Context()),
		Headers:   r.Header,
	}
}

func logChatError(r *http.Request, message, urlPath, model string, err error) {
	slog.ErrorContext(r.Context(), message,
		"component", "chat",
		"error", err,
		slog.Group("data",
			slog.String("endpoint", urlPath),
			slog.String("model", model),
		),
	)
}
