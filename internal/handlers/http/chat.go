package http

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sashabaranov/go-openai"
)

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
	debug            bool
	includeReasoning bool
}

func NewStreamHandler(apiKey, baseURL, model, provider, systemPrompt string, tools []openai.Tool, debug, includeReasoning bool) StreamHandler {
	return StreamHandler{
		apiKey:           apiKey,
		baseURL:          baseURL,
		model:            model,
		provider:         provider,
		systemPrompt:     systemPrompt,
		tools:            tools,
		debug:            debug,
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

	openaiReq := buildOpenAIRequest(h.model, h.provider, h.systemPrompt, h.tools, req.Messages, h.debug, h.includeReasoning)

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

	if h.debug {
		streamWithUsageLogging(resp.Body, w, flusher)
	} else {
		streamSimple(resp.Body, w, flusher)
	}
}

func streamSimple(src io.Reader, dst io.Writer, flusher http.Flusher) {
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			dst.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			if isUnexpectedStreamError(err) {
				slog.Error("stream read error", "error", err)
			}
			return
		}
	}
}

func streamWithUsageLogging(src io.Reader, dst io.Writer, flusher http.Flusher) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		line := scanner.Text()
		dst.Write([]byte(line + "\n"))
		flusher.Flush()

		if usage := extractUsage(line); usage != nil {
			logUsage(usage)
		}
	}
	if err := scanner.Err(); err != nil && isUnexpectedStreamError(err) {
		slog.Error("stream read error", "error", err)
	}
}

type systemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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

func buildOpenAIRequest(model, provider, systemPrompt string, tools []openai.Tool, messages []json.RawMessage, debug, includeReasoning bool) openAIRequest {
	req := openAIRequest{
		Model:    model,
		Messages: prependSystemMessage(systemPrompt, messages),
		Tools:    tools,
		Stream:   true,
	}
	if debug {
		req.Usage = &usageRequest{Include: true}
	}
	if provider != "" {
		req.Provider = &providerPreference{Only: []string{provider}}
	}
	req.IncludeReasoning = &includeReasoning
	return req
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

func logUsage(u *usageResponse) {
	slog.Info("usage",
		"prompt_tokens", u.PromptTokens,
		"completion_tokens", u.CompletionTokens,
		"total_tokens", u.TotalTokens,
		"cache_discount", u.CacheDiscount,
	)
}
