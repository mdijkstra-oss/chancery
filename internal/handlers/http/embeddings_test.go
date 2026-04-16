package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hermes-logos/internal/prompts"
)

func TestEmbeddingsHandler(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		upstreamCode int
		upstreamBody string
		wantStatus   int
		wantContains string
	}{
		{
			name:         "happy path forwards upstream body",
			body:         `{"input":["hello","world"]}`,
			upstreamCode: http.StatusOK,
			upstreamBody: `{"data":[{"index":0,"embedding":[0.1]}],"usage":{"total_tokens":4}}`,
			wantStatus:   http.StatusOK,
			wantContains: `"embedding":[0.1]`,
		},
		{
			name:       "empty input rejected",
			body:       `{"input":[]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json rejected",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:         "upstream error yields 502",
			body:         `{"input":["x"]}`,
			upstreamCode: http.StatusInternalServerError,
			upstreamBody: `{"error":"boom"}`,
			wantStatus:   http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.upstreamCode)
				w.Write([]byte(tt.upstreamBody))
			}))
			defer srv.Close()

			cfg := prompts.PromptConfig{
				Model:      "text-embedding-3-large",
				Dimensions: 1024,
				Provider:   prompts.ProviderConfig{BaseURL: srv.URL, APIKey: "test-key"},
			}
			handler := NewEmbeddingsHandler(cfg)

			req := httptest.NewRequest(http.MethodPost, "/embeddings", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			handler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantContains != "" {
				got, _ := io.ReadAll(rec.Body)
				if !strings.Contains(string(got), tt.wantContains) {
					t.Errorf("body %q does not contain %q", got, tt.wantContains)
				}
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  int
	}{
		{name: "empty", input: nil, want: 0},
		{name: "single short", input: []string{"abcd"}, want: 1},
		{name: "multiple", input: []string{"abcdefgh", "ijklmnop"}, want: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := estimateTokens(tt.input); got != tt.want {
				t.Errorf("estimateTokens(%v) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
