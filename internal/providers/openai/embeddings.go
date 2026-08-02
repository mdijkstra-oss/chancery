package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/providers/httpx"
	"github.com/mdijkstra-oss/chancery/internal/ratelimit"
)

type EmbedRequest struct {
	Input      []string `json:"input"`
	Model      string   `json:"model"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type EmbedUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type embedResponsePeek struct {
	Usage EmbedUsage `json:"usage"`
}

type EmbedResult struct {
	Body        []byte
	TotalTokens int
}

func Embed(ctx context.Context, input []string, model string, dimensions int, provider prompts.ProviderConfig) (EmbedResult, error) {
	body, err := json.Marshal(EmbedRequest{Input: input, Model: model, Dimensions: dimensions})
	if err != nil {
		return EmbedResult{}, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return EmbedResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	resp, err := httpx.Client.Do(req)
	if err != nil {
		if httpx.IsConnectTimeout(err) {
			return EmbedResult{}, ratelimit.Retryable(fmt.Errorf("openai embed: connect timeout: %w", err))
		}
		return EmbedResult{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return EmbedResult{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		err := fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(respBody))
		if d := ratelimit.ParseRetryAfterHeader(resp.Header.Get("Retry-After")); d > 0 {
			return EmbedResult{}, ratelimit.RetryableWithDelay(err, d)
		}
		return EmbedResult{}, ratelimit.Retryable(err)
	}
	if resp.StatusCode != http.StatusOK {
		return EmbedResult{}, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var peek embedResponsePeek
	json.Unmarshal(respBody, &peek)
	return EmbedResult{Body: respBody, TotalTokens: peek.Usage.TotalTokens}, nil
}
