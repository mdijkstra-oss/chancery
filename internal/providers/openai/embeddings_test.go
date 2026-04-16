package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hermes-logos/internal/prompts"
)

func TestEmbed(t *testing.T) {
	tests := []struct {
		name         string
		input        []string
		model        string
		dimensions   int
		upstreamCode int
		upstreamBody string
		wantErr      bool
		checkRequest func(t *testing.T, gotBody EmbedRequest, gotAuth string, gotURLPath string)
		checkResult  func(t *testing.T, res EmbedResult)
	}{
		{
			name:         "happy path returns body and parses usage",
			input:        []string{"hello", "world"},
			model:        "text-embedding-3-large",
			dimensions:   1024,
			upstreamCode: http.StatusOK,
			upstreamBody: `{"data":[{"index":0,"embedding":[0.1,0.2]},{"index":1,"embedding":[0.3,0.4]}],"usage":{"prompt_tokens":4,"total_tokens":4}}`,
			checkRequest: func(t *testing.T, got EmbedRequest, auth, path string) {
				if path != "/embeddings" {
					t.Errorf("path = %q, want /embeddings", path)
				}
				if auth != "Bearer test-key" {
					t.Errorf("auth = %q", auth)
				}
				if got.Model != "text-embedding-3-large" {
					t.Errorf("model = %q", got.Model)
				}
				if got.Dimensions != 1024 {
					t.Errorf("dimensions = %d", got.Dimensions)
				}
				if len(got.Input) != 2 || got.Input[0] != "hello" {
					t.Errorf("input = %v", got.Input)
				}
			},
			checkResult: func(t *testing.T, res EmbedResult) {
				if res.TotalTokens != 4 {
					t.Errorf("total tokens = %d, want 4", res.TotalTokens)
				}
				if !strings.Contains(string(res.Body), `"embedding":[0.1,0.2]`) {
					t.Errorf("body does not contain embedding: %s", res.Body)
				}
			},
		},
		{
			name:         "dimensions zero omitted from request",
			input:        []string{"a"},
			model:        "text-embedding-3-small",
			dimensions:   0,
			upstreamCode: http.StatusOK,
			upstreamBody: `{"data":[],"usage":{"total_tokens":1}}`,
			checkRequest: func(t *testing.T, got EmbedRequest, auth, path string) {
				if got.Dimensions != 0 {
					t.Errorf("dimensions = %d, want 0", got.Dimensions)
				}
			},
			checkResult: func(t *testing.T, res EmbedResult) {
				if res.TotalTokens != 1 {
					t.Errorf("total tokens = %d", res.TotalTokens)
				}
			},
		},
		{
			name:         "non-200 upstream returns error",
			input:        []string{"a"},
			model:        "text-embedding-3-large",
			upstreamCode: http.StatusTooManyRequests,
			upstreamBody: `{"error":"rate limited"}`,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody EmbedRequest
			var gotAuth, gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				gotPath = r.URL.Path
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &gotBody)
				w.WriteHeader(tt.upstreamCode)
				w.Write([]byte(tt.upstreamBody))
			}))
			defer srv.Close()

			provider := prompts.ProviderConfig{BaseURL: srv.URL, APIKey: "test-key"}
			res, err := Embed(context.Background(), tt.input, tt.model, tt.dimensions, provider)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkRequest != nil {
				tt.checkRequest(t, gotBody, gotAuth, gotPath)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, res)
			}
		})
	}
}
