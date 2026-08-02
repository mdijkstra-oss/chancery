package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/mdijkstra-oss/chancery/internal/auth"
	"github.com/mdijkstra-oss/chancery/internal/fn"
	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/ratelimit"
	"github.com/mdijkstra-oss/chancery/internal/responses"

	"github.com/go-chi/chi/v5"
)

// NewChatHandler serves every agent route. forwardHeaders is the operator's allowlist,
// copied onto the outbound request so the backend can record who a request came from.
func NewChatHandler(
	registry prompts.Registry,
	client *responses.Client,
	limiter *ratelimit.Limiter,
	forwardHeaders []string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, registry, client, limiter, forwardHeaders)
	}
}

func handleChat(
	w http.ResponseWriter,
	r *http.Request,
	registry prompts.Registry,
	client *responses.Client,
	limiter *ratelimit.Limiter,
	forwardHeaders []string,
) {
	urlPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	resolved, err := registry.ResolveAgent(urlPath)
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
	applyQueryOverrides(&agent, r.URL.Query())
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

	request := responses.Request{Body: composed, Identity: identityFor(r, urlPath, forwardHeaders)}
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

// A query parameter names a field of the body it overrides, so a caller that cannot
// edit the body it sends can still set one per request.
func applyQueryOverrides(agent *responses.Agent, query url.Values) {
	if choice := query.Get("tool_choice"); choice != "" {
		agent.ToolChoice = choice
	}
	if summary := query.Get("reasoning_summary"); summary != "" {
		agent.ReasoningSummary = summary
	}
}

// Identity is what chancery knows about a request and nothing it was trusted with: the
// caller's bearer token is consumed by the middleware and never reaches here.
func identityFor(r *http.Request, urlPath string, forwardHeaders []string) responses.Identity {
	headers := fn.Map(forwardHeaders, func(name string) responses.Header {
		return responses.Header{Name: name, Values: r.Header.Values(name)}
	})
	return responses.Identity{
		RequestID: RequestIDFromContext(r.Context()),
		Agent:     urlPath,
		Subject:   auth.UserFromContext(r.Context()),
		Headers:   fn.Filter(headers, hasValues),
	}
}

func hasValues(header responses.Header) bool {
	return len(header.Values) > 0
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
