package http

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// {
//   "model": "anthropic/claude-haiku-4.5",
//   "messages": [
//     {
//       "role": "system",
//       "content": "system prompt + tools definitions from files",
//       "cache_control": {"type": "ephemeral"}  // <-- CACHE BREAKPOINT
//     },
//     {"role": "user", "content": "hello"},
//     {"role": "assistant", "content": "hi there"},
//     {"role": "user", "content": "what's 2+2?"}
//   ]
// }

type ChatRequest struct {
	Messages []json.RawMessage `json:"messages"`
}

type StreamHandler struct {
	apiKey           string
	baseURL          string
	model            string
	provider         string
	systemPrompt     string
	tools            []openai.Tool
	verbose          bool
	includeReasoning bool
	cacheInterval    int
	maxTokenWindow   int
}

func NewStreamHandler(apiKey, baseURL, model, provider, systemPrompt string, tools []openai.Tool, verbose, includeReasoning bool, cacheInterval, maxTokenWindow int) StreamHandler {
	return StreamHandler{
		apiKey:           apiKey,
		baseURL:          baseURL,
		model:            model,
		provider:         provider,
		systemPrompt:     systemPrompt,
		tools:            tools,
		verbose:          verbose,
		includeReasoning: includeReasoning,
		cacheInterval:    cacheInterval,
		maxTokenWindow:   maxTokenWindow,
	}
}

func (h StreamHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		http.Error(w, "messages required", http.StatusBadRequest)
		return
	}

	enableCaching := shouldEnablePromptCaching(h.model)
	messagesWithCache, breakpoints := addCacheBreakpoints(req.Messages, enableCaching, h.cacheInterval)

	openaiReq := buildOpenAIRequest(h.model, h.provider, h.systemPrompt, h.tools, messagesWithCache, h.includeReasoning, enableCaching)

	breakdown := calculateTokenBreakdown(openaiReq, h.systemPrompt)
	totalEstimated := sumTokenBreakdown(breakdown)

	if h.maxTokenWindow > 0 && totalEstimated > h.maxTokenWindow {
		http.Error(w, "Out of context, please reload", http.StatusInternalServerError)
		return
	}

	logOutgoingRequest(openaiReq, breakdown, breakpoints, enableCaching, h.verbose)

	proxyReq, err := http.NewRequestWithContext(r.Context(), "POST", h.baseURL+"/chat/completions", jsonReader(openaiReq))
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+h.apiKey)

	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		slog.Error("upstream request failed", "error", err)
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("upstream error", "status", resp.StatusCode, "body", string(body))
		http.Error(w, string(body), resp.StatusCode)
		return
	}

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, resp.Body)
		return
	}

	streamWithUsageLogging(resp.Body, w, flusher, breakdown, breakpoints, enableCaching, len(openaiReq.Tools) > 0, h.verbose)
}

func streamWithUsageLogging(src io.Reader, dst io.Writer, flusher http.Flusher, breakdown map[string]int, breakpoints []cacheBreakpointInfo, enableCaching, hasTools, verbose bool) {
	scanner := bufio.NewScanner(src)
	lineCount := 0
	var collected strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		lineWithNewline := line + "\n"
		dst.Write([]byte(lineWithNewline))
		flusher.Flush()
		lineCount++

		if verbose {
			collected.WriteString(lineWithNewline)
		}

		if usage := extractUsage(line); usage != nil {
			logUsage(usage, breakdown, breakpoints, enableCaching, hasTools)
		}
	}

	if err := scanner.Err(); err != nil && isUnexpectedStreamError(err) {
		slog.Error("stream read error", "error", err)
	} else {
		slog.Info("stream_complete", "lines_received", lineCount)
		if verbose {
			slog.Info("raw_response", "data", collected.String())
		}
	}
}

type cacheControl struct {
	Type string `json:"type"`
}

type systemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type toolWithCache struct {
	openai.Tool
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type openAIRequest struct {
	Model            string            `json:"model"`
	Messages         []json.RawMessage `json:"messages"`
	Tools            []toolWithCache   `json:"tools,omitempty"`
	Stream           bool              `json:"stream"`
	Usage            *usageRequest     `json:"usage,omitempty"`
	Provider         *providerPreference `json:"provider,omitempty"`
	IncludeReasoning *bool             `json:"include_reasoning,omitempty"`
}

type usageRequest struct {
	Include bool `json:"include"`
}

type providerPreference struct {
	Only []string `json:"only"`
}

func buildOpenAIRequest(model, provider, systemPrompt string, tools []openai.Tool, messages []json.RawMessage, includeReasoning, enableCaching bool) openAIRequest {
	req := openAIRequest{
		Model:    model,
		Messages: prependSystemMessage(systemPrompt, messages),
		Tools:    wrapToolsWithCache(tools, enableCaching),
		Stream:   true,
		Usage:    &usageRequest{Include: true},
	}
	if provider != "" {
		req.Provider = &providerPreference{Only: []string{provider}}
	}
	req.IncludeReasoning = &includeReasoning
	return req
}

func shouldEnablePromptCaching(model string) bool {
	return strings.Contains(model, "claude") || strings.Contains(model, "anthropic")
}

func wrapToolsWithCache(tools []openai.Tool, enableCaching bool) []toolWithCache {
	wrapped := make([]toolWithCache, len(tools))
	for i, tool := range tools {
		wrapped[i] = toolWithCache{Tool: tool}
	}
	if enableCaching && len(wrapped) > 0 {
		wrapped[len(wrapped)-1].CacheControl = &cacheControl{Type: "ephemeral"}
	}
	return wrapped
}

type messageWithCache struct {
	Role         string        `json:"role"`
	Content      string        `json:"content,omitempty"`
	ToolCalls    interface{}   `json:"tool_calls,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type cacheBreakpointInfo struct {
	messageIndex int
	tokenPos     int
	breakpointNum int
}

func addCacheBreakpoints(messages []json.RawMessage, enableCaching bool, cacheInterval int) ([]json.RawMessage, []cacheBreakpointInfo) {
	if !enableCaching || cacheInterval <= 0 {
		return messages, nil
	}

	result := make([]json.RawMessage, len(messages))
	accumulated := 0
	nextThreshold := cacheInterval
	breakpointsUsed := 0
	maxBreakpoints := 3
	var breakpoints []cacheBreakpointInfo

	for i, rawMsg := range messages {
		var msg map[string]interface{}
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			result[i] = rawMsg
			continue
		}

		content, _ := msg["content"].(string)
		msgTokens := estimateTokens(content)

		if toolCalls, ok := msg["tool_calls"].([]interface{}); ok {
			toolCallsJSON, _ := json.Marshal(toolCalls)
			msgTokens += estimateTokens(string(toolCallsJSON))
		}

		accumulated += msgTokens

		if accumulated >= nextThreshold && breakpointsUsed < maxBreakpoints {
			msgWithCache := messageWithCache{
				CacheControl: &cacheControl{Type: "ephemeral"},
			}

			if role, ok := msg["role"].(string); ok {
				msgWithCache.Role = role
			}
			if c, ok := msg["content"].(string); ok {
				msgWithCache.Content = c
			}
			if tc, ok := msg["tool_calls"]; ok {
				msgWithCache.ToolCalls = tc
			}
			if tid, ok := msg["tool_call_id"].(string); ok {
				msgWithCache.ToolCallID = tid
			}

			msgJSON, _ := json.Marshal(msgWithCache)
			result[i] = msgJSON

			breakpoints = append(breakpoints, cacheBreakpointInfo{
				messageIndex: i,
				tokenPos:     accumulated,
				breakpointNum: breakpointsUsed + 2,
			})

			breakpointsUsed++
			nextThreshold += cacheInterval
		} else {
			result[i] = rawMsg
		}
	}

	return result, breakpoints
}

func prependSystemMessage(systemPrompt string, messages []json.RawMessage) []json.RawMessage {
	sysMsg := systemMessage{Role: "system", Content: systemPrompt}
	sysMsgJSON, _ := json.Marshal(sysMsg)
	return append([]json.RawMessage{sysMsgJSON}, messages...)
}

func jsonReader(v any) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		pw.CloseWithError(json.NewEncoder(pw).Encode(v))
	}()
	return pr
}

func isUnexpectedStreamError(err error) bool {
	return err != io.EOF && !errors.Is(err, context.Canceled)
}

var proxyHeaders = []string{"Content-Type", "Cache-Control"}

func copyHeaders(dst, src http.Header) {
	for _, h := range proxyHeaders {
		if v := src.Get(h); v != "" {
			dst.Set(h, v)
		}
	}
}

type streamChunk struct {
	Usage *usageResponse `json:"usage,omitempty"`
}

type usageResponse struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CacheDiscount    float64 `json:"cache_discount,omitempty"`
}

func extractUsage(line string) *usageResponse {
	if !strings.HasPrefix(line, "data: ") {
		return nil
	}
	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return nil
	}
	var chunk streamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil
	}
	return chunk.Usage
}

func formatRatio(prompt, completion int) string {
	if completion == 0 {
		return "0:1"
	}
	ratio := math.Round(float64(prompt) / float64(completion))
	return fmt.Sprintf("%.0f:1", ratio)
}

func calculateCachedTokens(breakdown map[string]int, breakpoints []cacheBreakpointInfo, enableCaching, hasTools bool) int {
	if !enableCaching {
		return 0
	}

	if len(breakpoints) == 0 {
		if hasTools {
			return breakdown["system"] + breakdown["tool_defs"]
		}
		return 0
	}

	maxPos := 0
	for _, bp := range breakpoints {
		if bp.tokenPos > maxPos {
			maxPos = bp.tokenPos
		}
	}
	return maxPos
}

func logUsage(u *usageResponse, breakdown map[string]int, breakpoints []cacheBreakpointInfo, enableCaching, hasTools bool) {
	ratio := formatRatio(u.PromptTokens, u.CompletionTokens)
	totalEstimate := sumTokenBreakdown(breakdown)
	estCached := calculateCachedTokens(breakdown, breakpoints, enableCaching, hasTools)
	estUncached := totalEstimate - estCached

	attrs := []any{
		"prompt_tokens", u.PromptTokens,
		"completion_tokens", u.CompletionTokens,
		"total_tokens", u.TotalTokens,
		"input_output_ratio", ratio,
		"cache_discount", u.CacheDiscount,
		"est_system", breakdown["system"],
		"est_tool_defs", breakdown["tool_defs"],
		"est_user_msgs", breakdown["user_msgs"],
		"est_assistant_msgs", breakdown["assistant_msgs"],
		"est_tool_calls", breakdown["tool_calls"],
		"est_tool_responses", breakdown["tool_responses"],
		"est_total", totalEstimate,
		"est_cached", estCached,
		"est_uncached", estUncached,
	}

	if enableCaching && hasTools {
		attrs = append(attrs, "bp1", "tools")
	}

	for _, bp := range breakpoints {
		attrs = append(attrs, fmt.Sprintf("bp%d", bp.breakpointNum), bp.tokenPos)
	}

	slog.Info("usage", attrs...)
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func sumTokenBreakdown(breakdown map[string]int) int {
	total := 0
	for _, v := range breakdown {
		total += v
	}
	return total
}

func calculateTokenBreakdown(req openAIRequest, systemPrompt string) map[string]int {
	breakdown := map[string]int{
		"system":         estimateTokens(systemPrompt),
		"tool_defs":      0,
		"user_msgs":      0,
		"assistant_msgs": 0,
		"tool_calls":     0,
		"tool_responses": 0,
	}

	toolsJSON, _ := json.Marshal(req.Tools)
	breakdown["tool_defs"] = estimateTokens(string(toolsJSON))

	for _, rawMsg := range req.Messages {
		var msg map[string]interface{}
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			continue
		}

		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)

		switch role {
		case "system":
			continue
		case "user":
			breakdown["user_msgs"] += estimateTokens(content)
		case "assistant":
			breakdown["assistant_msgs"] += estimateTokens(content)
			if toolCalls, ok := msg["tool_calls"].([]interface{}); ok {
				toolCallsJSON, _ := json.Marshal(toolCalls)
				breakdown["tool_calls"] += estimateTokens(string(toolCallsJSON))
			}
		case "tool":
			breakdown["tool_responses"] += estimateTokens(content)
		}
	}

	return breakdown
}

func logOutgoingRequest(req openAIRequest, breakdown map[string]int, breakpoints []cacheBreakpointInfo, enableCaching bool, verbose bool) {
	totalEstimate := sumTokenBreakdown(breakdown)

	slog.Info("outgoing_request",
		"model", req.Model,
		"message_count", len(req.Messages),
		"tool_count", len(req.Tools),
		"stream", req.Stream,
		"est_system", breakdown["system"],
		"est_tool_defs", breakdown["tool_defs"],
		"est_user_msgs", breakdown["user_msgs"],
		"est_assistant_msgs", breakdown["assistant_msgs"],
		"est_tool_calls", breakdown["tool_calls"],
		"est_tool_responses", breakdown["tool_responses"],
		"est_total", totalEstimate,
	)

	if verbose {
		reqJSON, _ := json.Marshal(req)
		slog.Info("raw_outgoing_request", "data", string(reqJSON))
	}
}
