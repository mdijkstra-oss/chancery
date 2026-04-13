package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"hermes-logos/internal/prompts"
)

func NewChatHandler(inspect bool, registry prompts.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleChat(w, r, inspect, registry)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request, inspect bool, registry prompts.Registry) {
	ctx := r.Context()
	urlPath := strings.TrimPrefix(chi.URLParam(r, "*"), "/")

	agent, ok := registry.Agents[urlPath]
	if !ok {
		http.Error(w, "unknown agent: "+urlPath, http.StatusNotFound)
		return
	}

	promptCfg, ok := registry.Configs[urlPath]
	if !ok {
		http.Error(w, "no config for agent: "+urlPath, http.StatusInternalServerError)
		return
	}

	req, err := decodeRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.Messages = ExpandMessages(req.Messages, registry.Modes)
	req.Messages = ExpandApproaches(req.Messages, registry.Approaches.Entries)

	model := promptCfg.Model
	reasoningEffort := promptCfg.ReasoningEffort
	verbosity := promptCfg.Verbosity

	toolNames := ExtractToolNames(req.Tools)

	toolPrompt, _, err := prompts.LoadToolPrompts(filepath.Join(prompts.PromptsDir, "tools"), toolNames)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fullPrompt := agent.Prompt
	if toolPrompt != "" {
		fullPrompt = fullPrompt + "\n\n" + toolPrompt
	}

	toolChoice := r.URL.Query().Get("tool_choice")
	temperature := parseTemperature(r.URL.Query().Get("temperature"))
	if temperature == nil {
		temperature = promptCfg.Temperature
	}
	reasoningSummary := r.URL.Query().Get("reasoning_summary")
	if reasoningSummary == "" {
		reasoningSummary = promptCfg.ReasoningSummary
	}

	provider := promptCfg.Provider
	proto := MustProtocol(provider.Protocol)

	params := RequestParams{
		Model:            model,
		SystemPrompt:     fullPrompt,
		ReasoningEffort:  reasoningEffort,
		ReasoningSummary: reasoningSummary,
		Verbosity:        verbosity,
		ServiceTier:      promptCfg.ServiceTier,
		ToolChoice:       toolChoice,
		Temperature:      temperature,
		Tools:            req.Tools,
		Messages:         req.Messages,
		ResponseFormat:   req.ResponseFormat,
	}
	apiReq := proto.BuildRequest(params)

	trigger := deriveTrigger(req.Messages)

	forceCompact := r.URL.Query().Get("compact") == "true"
	tokens := estimateTokens(params.Messages)
	if forceCompact || shouldCompact(promptCfg.CompactAt, tokens) {
		compacterAgent, compacterOk := registry.Agents["compacter"]
		compacterCfg, compacterCfgOk := registry.Configs["compacter"]
		if !compacterOk || !compacterCfgOk {
			slog.ErrorContext(ctx, "compacter agent not found in registry", "component", "chat")
		} else {
			if compacterCfg.Provider.Protocol != prompts.ProtocolResponses {
				panic("compacter must use responses protocol")
			}
			compactReq := buildCompactRequest(compacterCfg.Model, compacterAgent.Prompt, stripForCompaction(req.Messages))
			start := time.Now()
			resp, err := proxyWithRetry(ctx, compactReq, compacterCfg.Provider, "/responses")
			if err != nil {
				handleProxyError(ctx, w, err)
				return
			}
			defer resp.Body.Close()
			if isErrorResponse(resp) {
				handleUpstreamError(ctx, w, resp)
				return
			}
			slog.InfoContext(ctx, "compaction started, context exceeds token limit", "component", "chat", slog.Group("data", slog.Int("tokens", tokens), slog.Int("compact_at", promptCfg.CompactAt)))
			if inspect {
				inspectJSON("compacter request", compactReq)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			flusher := w.(http.Flusher)
			usage, err := streamCompaction(resp.Body, w, flusher)
			if err != nil {
				slog.ErrorContext(ctx, "compaction stream failed", "component", "chat", "error", err)
				return
			}
			durationMs := time.Since(start).Milliseconds()
			rec := buildCallRecord("compacter", compacterCfg.Model, "", "", trigger, usage, compacterCfg.Pricing, durationMs)
			logCallRecord(ctx, rec)
			return
		}
	}

	if inspect {
		inspectJSON(urlPath+" request", apiReq)
	}

	start := time.Now()
	resp, err := proxyWithRetry(ctx, apiReq, provider, proto.URLPath)
	if err != nil {
		handleProxyError(ctx, w, err)
		return
	}
	defer resp.Body.Close()

	if isErrorResponse(resp) {
		handleUpstreamError(ctx, w, resp)
		return
	}

	usage := streamResponse(ctx, w, resp, proto, inspect, urlPath)
	durationMs := time.Since(start).Milliseconds()
	rec := buildCallRecord(urlPath, model, reasoningEffort, promptCfg.ServiceTier, trigger, usage, promptCfg.Pricing, durationMs)
	logCallRecord(ctx, rec)
}

func decodeRequest(r *http.Request) (ChatRequest, error) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, err
	}
	if len(req.Messages) == 0 {
		return req, http.ErrBodyNotAllowed
	}
	return req, nil
}

var errRateLimited = errors.New("rate limit exceeded after retries")

const maxUpstreamRetries = 3

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	base := time.Duration(1<<uint(attempt)) * time.Second
	jitter := time.Duration(rand.Int64N(int64(time.Second)))
	return base + jitter
}

func proxyRequest(ctx context.Context, apiReq any, provider prompts.ProviderConfig, urlPath string) (*http.Response, error) {
	proxyReq, err := http.NewRequestWithContext(ctx, "POST", provider.BaseURL+urlPath, jsonReader(apiReq))
	if err != nil {
		return nil, err
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+provider.APIKey)

	return http.DefaultClient.Do(proxyReq)
}

func proxyWithRetry(ctx context.Context, apiReq any, provider prompts.ProviderConfig, urlPath string) (*http.Response, error) {
	for attempt := range maxUpstreamRetries {
		if err := acquireUpstream(ctx); err != nil {
			return nil, err
		}
		resp, err := proxyRequest(ctx, apiReq, provider, urlPath)
		releaseUpstream()

		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		resp.Body.Close()

		if attempt == maxUpstreamRetries-1 {
			return nil, errRateLimited
		}

		delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
		slog.WarnContext(ctx, "rate limited by upstream, retrying with backoff", "component", "chat", slog.Group("data", slog.Int("attempt", attempt+1), slog.Duration("delay", delay)))

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, errRateLimited
}

func handleProxyError(ctx context.Context, w http.ResponseWriter, err error) {
	if errors.Is(err, errRateLimited) {
		slog.ErrorContext(ctx, "rate limit retries exhausted", "component", "chat")
		http.Error(w, "Rate limited after retries", http.StatusTooManyRequests)
		return
	}
	slog.ErrorContext(ctx, "upstream request failed", "component", "chat", "error", err)
	http.Error(w, "upstream request failed", http.StatusBadGateway)
}

func isErrorResponse(resp *http.Response) bool {
	return resp.StatusCode >= 400
}

func handleUpstreamError(ctx context.Context, w http.ResponseWriter, resp *http.Response) {
	body, _ := io.ReadAll(resp.Body)
	slog.ErrorContext(ctx, "upstream returned error response", "component", "chat", slog.Group("data", slog.Int("status", resp.StatusCode), slog.String("body", string(body))))
	http.Error(w, string(body), resp.StatusCode)
}

func streamResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response, proto ProtocolFuncs, inspect bool, endpoint string) *UsageResponse {
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return nil
	}

	return proto.ForwardStream(ctx, resp.Body, w, flusher, inspect, endpoint)
}

func parseTemperature(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
