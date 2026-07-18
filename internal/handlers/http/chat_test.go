package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/quota"
	"hermes-logos/internal/ratelimit"

	"github.com/go-chi/chi/v5"
)

func TestChatQuotaDenial(t *testing.T) {
	var got quota.ReserveRequest
	quotaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode reservation: %v", err)
		}
		json.NewEncoder(w).Encode(quota.Reservation{Allowed: false, Reason: "daily_model_limit", RetryAfterSeconds: 60})
	}))
	defer quotaServer.Close()

	registry := prompts.Registry{
		Root:   t.TempDir(),
		Agents: map[string]prompts.CompiledAgent{"agent": {Prompt: "system prompt"}},
		Configs: map[string]prompts.PromptConfig{
			"agent": {Model: "model-a", MaxTokens: 100, Provider: prompts.ProviderConfig{Key: "provider-a"}},
		},
		Modes: map[string]string{},
	}
	quotaClient := quota.NewClient(quota.Config{ReserveURL: quotaServer.URL, SettleURL: quotaServer.URL, Timeout: time.Second})
	handler := NewChatHandler(registry, ratelimit.NewLimiter(), quotaClient)
	router := chi.NewRouter()
	router.Use(RequestContext(nil))
	router.Post("/*", handler)

	request := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(`{"messages":[{"role":"user","content":"hello"}]}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d; body: %s", response.Code, http.StatusTooManyRequests, response.Body.String())
	}
	if response.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q, want 60", response.Header().Get("Retry-After"))
	}
	if got.RequestID == "" {
		t.Error("request_id is empty")
	}
	if got.Subject != "" {
		t.Errorf("subject = %q, want anonymous", got.Subject)
	}
	if got.Endpoint != "agent" || got.Provider != "provider-a" || got.Model != "model-a" {
		t.Errorf("reservation facts = %#v", got)
	}
}
