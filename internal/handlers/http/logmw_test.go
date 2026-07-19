package http

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matthijn/hermes-logos/internal/logging"
)

func TestRequestContext(t *testing.T) {
	tests := []struct {
		name          string
		headers       []string
		requestValues map[string]string
		want          map[string]string
		wantTruncated bool
	}{
		{
			name:          "configured headers logged",
			headers:       []string{"X-Session-ID", "X-Tenant-ID"},
			requestValues: map[string]string{"X-Session-ID": "session-1", "X-Tenant-ID": "tenant-2", "Authorization": "Bearer secret"},
			want:          map[string]string{"x-session-id": "session-1", "x-tenant-id": "tenant-2"},
		},
		{
			name:          "absent headers omitted",
			headers:       []string{"X-Session-ID"},
			requestValues: map[string]string{"X-Other": "value"},
			want:          map[string]string{},
		},
		{
			name:          "oversized value bounded",
			headers:       []string{"X-Session-ID"},
			requestValues: map[string]string{"X-Session-ID": strings.Repeat("a", maxLoggedHeaderValueLength+10)},
			want:          map[string]string{"x-session-id": strings.Repeat("a", maxLoggedHeaderValueLength)},
			wantTruncated: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var attrs []slog.Attr
			var requestID string
			handler := RequestContext(test.headers)(captureAttrsHandler(&attrs, &requestID))
			req := httptest.NewRequest(http.MethodPost, "/agent", nil)
			for key, value := range test.requestValues {
				req.Header.Set(key, value)
			}
			handler.ServeHTTP(httptest.NewRecorder(), req)
			if contextAttr(attrs, "request_id") == "" {
				t.Error("logged request_id is empty")
			}
			if requestID == "" {
				t.Error("context request_id is empty")
			}
			got, truncated := headerAttrs(attrs)
			if len(got) != len(test.want) {
				t.Errorf("headers length = %d, want %d", len(got), len(test.want))
			}
			for key, want := range test.want {
				if got[key] != want {
					t.Errorf("header %s = %q, want %q", key, got[key], want)
				}
			}
			if truncated != test.wantTruncated {
				t.Errorf("truncated = %v, want %v", truncated, test.wantTruncated)
			}
		})
	}
}

func captureAttrsHandler(destination *[]slog.Attr, requestID *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*destination = logging.AttrsFromContext(r.Context())
		*requestID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
}

func contextAttr(attrs []slog.Attr, key string) string {
	for _, attr := range attrs {
		if attr.Key == key {
			return attr.Value.String()
		}
	}
	return ""
}

func headerAttrs(attrs []slog.Attr) (map[string]string, bool) {
	values := make(map[string]string)
	truncated := false
	for _, attr := range attrs {
		if attr.Key != "headers" {
			continue
		}
		for _, header := range attr.Value.Group() {
			if header.Key == "truncated" {
				truncated = header.Value.Bool()
				continue
			}
			values[header.Key] = header.Value.String()
		}
	}
	return values, truncated
}
