package http

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"hermes-logos/internal/logging"
)

type headerMapping struct {
	Header string
	LogKey string
}

var trackedHeaders = []headerMapping{
	{"X-Session-ID", "session_id"},
	{"X-Project-ID", "project_id"},
}

func RequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = logging.WithAttr(ctx, "request_id", generateRequestID())
		for _, h := range trackedHeaders {
			if v := r.Header.Get(h.Header); v != "" {
				ctx = logging.WithAttr(ctx, h.LogKey, v)
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
