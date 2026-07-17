package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func chatRouteHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusAccepted)
}

func embeddingsRouteHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func TestSetupRoutes(t *testing.T) {
	r := chi.NewRouter()
	SetupRoutes(r, chatRouteHandler, embeddingsRouteHandler, []string{"*"})

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"guidance listing absent", http.MethodGet, "/guidance", http.StatusMethodNotAllowed},
		{"guidance path remains an agent path", http.MethodPost, "/guidance", http.StatusAccepted},
		{"embeddings route", http.MethodPost, "/embeddings", http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)
			if res.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", res.Code, tt.wantStatus)
			}
		})
	}
}
