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
}

func NewStreamHandler(apiKey, baseURL, model, provider, systemPrompt string, tools []openai.Tool, verbose, includeReasoning bool) StreamHandler {
	return StreamHandler{
		apiKey:           apiKey,
		baseURL:          baseURL,
		model:            model,
		provider:         provider,
		systemPrompt:     systemPrompt,
		tools:            tools,
		verbose:          verbose,
		includeReasoning: includeReasoning,
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

	openaiReq := buildOpenAIRequest(h.model, h.provider, h.systemPrompt, h.tools, req.Messages, h.includeReasoning)

	breakdown := calculateTokenBreakdown(openaiReq, h.systemPrompt)
	logOutgoingRequest(openaiReq, breakdown, h.verbose)

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

	streamWithUsageLogging(resp.Body, w, flusher, breakdown, h.verbose)
}

func streamWithUsageLogging(src io.Reader, dst io.Writer, flusher http.Flusher, breakdown map[string]int, verbose bool) {
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
			logUsage(usage, breakdown)
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
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type openAIRequest struct {
	Model            string            `json:"model"`
	Messages         []json.RawMessage `json:"messages"`
	Tools            []openai.Tool     `json:"tools,omitempty"`
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

func buildOpenAIRequest(model, provider, systemPrompt string, tools []openai.Tool, messages []json.RawMessage, includeReasoning bool) openAIRequest {
	req := openAIRequest{
		Model:    model,
		Messages: prependSystemMessage(systemPrompt, shouldEnablePromptCaching(model), messages),
		Tools:    tools,
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

func prependSystemMessage(systemPrompt string, enableCaching bool, messages []json.RawMessage) []json.RawMessage {
	sysMsg := systemMessage{Role: "system", Content: systemPrompt}
	if enableCaching {
		sysMsg.CacheControl = &cacheControl{Type: "ephemeral"}
	}
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

func logUsage(u *usageResponse, breakdown map[string]int) {
	ratio := formatRatio(u.PromptTokens, u.CompletionTokens)
	totalEstimate := 0
	for _, v := range breakdown {
		totalEstimate += v
	}

	slog.Info("usage",
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
	)
}

func estimateTokens(text string) int {
	return len(text) / 4
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

func logOutgoingRequest(req openAIRequest, breakdown map[string]int, verbose bool) {
	totalEstimate := 0
	for _, v := range breakdown {
		totalEstimate += v
	}

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
