package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/matthijn/hermes-logos/internal/logging"
)

const maxLoggedHeaderValueLength = 512

type requestIDContextKey struct{}

func RequestContext(headers []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := generateRequestID()
			ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
			ctx = logging.WithAttr(ctx, "request_id", requestID)
			attrs := loggedHeaderAttrs(r, headers)
			if len(attrs) > 0 {
				ctx = logging.WithAttrs(ctx, slog.GroupAttrs("headers", attrs...))
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func loggedHeaderAttrs(r *http.Request, headers []string) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(headers)+1)
	truncated := false
	for _, header := range headers {
		value := strings.Join(r.Header.Values(header), ",")
		if value == "" {
			continue
		}
		if len(value) > maxLoggedHeaderValueLength {
			value = value[:maxLoggedHeaderValueLength]
			truncated = true
		}
		attrs = append(attrs, slog.String(strings.ToLower(header), value))
	}
	if truncated {
		attrs = append(attrs, slog.Bool("truncated", true))
	}
	return attrs
}

func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
