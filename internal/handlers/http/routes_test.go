package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"hermes-logos/internal/auth"

	"github.com/go-chi/chi/v5"
)

func chatRouteHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}

func embeddingsRouteHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func TestSetupRoutes(t *testing.T) {
	validator, err := auth.NewValidator(context.Background(), auth.Config{})
	if err != nil {
		t.Fatalf("NewValidator(): %v", err)
	}
	r := chi.NewRouter()
	SetupRoutes(r, chatRouteHandler, embeddingsRouteHandler, JWTAuthentication(validator), []string{"*"}, []string{"X-Session-ID"})

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"health route", http.MethodGet, "/health", http.StatusOK, "ok\n"},
		{"guidance listing absent", http.MethodGet, "/guidance", http.StatusMethodNotAllowed, ""},
		{"guidance path remains an agent path", http.MethodPost, "/guidance", http.StatusAccepted, ""},
		{"embeddings route", http.MethodPost, "/embeddings", http.StatusCreated, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)
			if res.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", res.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && res.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", res.Body.String(), tt.wantBody)
			}
		})
	}
}
