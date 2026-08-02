package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mdijkstra-oss/chancery/internal/auth"
	"github.com/mdijkstra-oss/chancery/internal/logging"

	"github.com/golang-jwt/jwt/v5"
)

const maxAuthorizationHeaderLength = 8192

type unauthorizedResponse struct {
	Error string `json:"error"`
}

func JWTAuthentication(validator auth.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !validator.Enabled() {
				ctx := logging.WithAttr(auth.WithUser(r.Context(), ""), "user", "")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			raw, reason := bearerToken(r)
			if reason != "" {
				rejectUnauthorized(w, r, reason)
				return
			}
			user, err := validator.Validate(r.Context(), raw)
			if err != nil {
				rejectUnauthorized(w, r, validationReason(err))
				return
			}
			ctx := logging.WithAttr(auth.WithUser(r.Context(), user), "user", user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) (string, string) {
	values := r.Header.Values("Authorization")
	if len(values) == 0 {
		return "", "missing_token"
	}
	if len(values) != 1 || len(values[0]) > maxAuthorizationHeaderLength {
		return "", "invalid_header"
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", "invalid_header"
	}
	return parts[1], ""
}

func validationReason(err error) string {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return "expired"
	case errors.Is(err, jwt.ErrTokenNotValidYet), errors.Is(err, jwt.ErrTokenUsedBeforeIssued):
		return "not_yet_valid"
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return "invalid_signature"
	case errors.Is(err, jwt.ErrTokenInvalidAudience):
		return "invalid_audience"
	case errors.Is(err, jwt.ErrTokenInvalidIssuer):
		return "invalid_issuer"
	case errors.Is(err, jwt.ErrTokenRequiredClaimMissing), errors.Is(err, jwt.ErrTokenInvalidSubject):
		return "invalid_subject"
	default:
		return "invalid_token"
	}
}

func rejectUnauthorized(w http.ResponseWriter, r *http.Request, reason string) {
	ctx := logging.WithAttr(auth.WithUser(r.Context(), ""), "user", "")
	slog.WarnContext(ctx, "authentication failed", "component", "auth", slog.Group("data", slog.String("reason", reason)))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(unauthorizedResponse{Error: "unauthorized"})
}
