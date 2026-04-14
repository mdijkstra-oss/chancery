package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
)

func BuildHTTPRequest(ctx context.Context, params protocol.RequestParams, provider prompts.ProviderConfig) (*http.Request, error) {
	body, err := json.Marshal(protocol.BuildResponsesRequestFromParams(params))
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	url := provider.BaseURL + "/responses"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	return req, nil
}
