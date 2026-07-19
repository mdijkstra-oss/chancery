package httpstream

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matthijn/hermes-logos/internal/ratelimit"
)

func TestOpen(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		retryAfter    string
		wantErr       bool
		wantRetryable bool
		wantDelay     time.Duration
	}{
		{name: "ok streams body", status: http.StatusOK},
		{name: "429 with retry-after is retryable with delay", status: http.StatusTooManyRequests, retryAfter: "2", wantErr: true, wantRetryable: true, wantDelay: 2 * time.Second},
		{name: "429 without retry-after is retryable", status: http.StatusTooManyRequests, wantErr: true, wantRetryable: true},
		{name: "500 is a plain error", status: http.StatusInternalServerError, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				if tt.retryAfter != "" {
					rw.Header().Set("Retry-After", tt.retryAfter)
				}
				rw.WriteHeader(tt.status)
				if tt.status == http.StatusOK {
					rw.Write([]byte("data: hello\n\n"))
				}
			}))
			defer server.Close()

			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			rec := httptest.NewRecorder()
			scanner, body, err := Open(rec, req, "test")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if got := ratelimit.IsRetryable(err); got != tt.wantRetryable {
					t.Fatalf("IsRetryable = %v, want %v", got, tt.wantRetryable)
				}
				if tt.wantDelay > 0 {
					if got := ratelimit.ExtractDelay(err); got != tt.wantDelay {
						t.Fatalf("ExtractDelay = %v, want %v", got, tt.wantDelay)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer body.Close()
			if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
				t.Fatalf("Content-Type = %q, want text/event-stream", ct)
			}
			if !scanner.Scan() {
				t.Fatalf("scanner produced no lines: %v", scanner.Err())
			}
			if got := scanner.Text(); got != "data: hello" {
				t.Fatalf("line = %q, want %q", got, "data: hello")
			}
		})
	}
}
