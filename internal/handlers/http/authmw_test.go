package http

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matthijn/hermes-logos/internal/auth"
	"github.com/matthijn/hermes-logos/internal/logging"

	"github.com/golang-jwt/jwt/v5"
)

type handlerResult struct {
	user    string
	logUser string
}

func TestJWTAuthentication(t *testing.T) {
	key := middlewareRSAKey(t)
	validator := middlewareValidator(t, key)
	disabled, err := auth.NewValidator(context.Background(), auth.Config{})
	if err != nil {
		t.Fatalf("disabled NewValidator(): %v", err)
	}
	now := time.Now()
	tests := []struct {
		name       string
		validator  auth.Validator
		token      string
		wantStatus int
		wantUser   string
	}{
		{
			name:       "valid token",
			validator:  validator,
			token:      middlewareToken(t, key, now.Add(time.Hour), "user-123"),
			wantStatus: http.StatusNoContent,
			wantUser:   "user-123",
		},
		{
			name:       "expired token",
			validator:  validator,
			token:      middlewareToken(t, key, now.Add(-time.Minute), "user-123"),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing header",
			validator:  validator,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "disabled mode",
			validator:  disabled,
			wantStatus: http.StatusNoContent,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := &handlerResult{}
			handler := JWTAuthentication(test.validator)(resultHandler(result))
			req := httptest.NewRequest(http.MethodPost, "/agent", nil)
			if test.token != "" {
				req.Header.Set("Authorization", "Bearer "+test.token)
			}
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", res.Code, test.wantStatus)
			}
			if test.wantStatus == http.StatusUnauthorized {
				assertUnauthorizedResponse(t, res)
				return
			}
			if result.user != test.wantUser {
				t.Errorf("context user = %q, want %q", result.user, test.wantUser)
			}
			if result.logUser != test.wantUser {
				t.Errorf("log user = %q, want %q", result.logUser, test.wantUser)
			}
		})
	}
}

func resultHandler(result *handlerResult) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result.user = auth.UserFromContext(r.Context())
		for _, attr := range logging.AttrsFromContext(r.Context()) {
			if attr.Key == "user" {
				result.logUser = attr.Value.String()
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func assertUnauthorizedResponse(t *testing.T, res *httptest.ResponseRecorder) {
	t.Helper()
	if res.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", res.Header().Get("Content-Type"))
	}
	if res.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("WWW-Authenticate = %q", res.Header().Get("WWW-Authenticate"))
	}
	var body unauthorizedResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "unauthorized" {
		t.Errorf("error = %q", body.Error)
	}
}

func middlewareValidator(t *testing.T, key *rsa.PrivateKey) auth.Validator {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey(): %v", err)
	}
	path := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}
	validator, err := auth.NewValidator(context.Background(), auth.Config{
		PublicKeyFile: path,
		Issuer:        "https://issuer.example/",
		Audience:      "hermes-logos",
		Algorithms:    []string{"RS256"},
	})
	if err != nil {
		t.Fatalf("NewValidator(): %v", err)
	}
	t.Cleanup(validator.Close)
	return validator
}

func middlewareRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(): %v", err)
	}
	return key
}

func middlewareToken(t *testing.T, key *rsa.PrivateKey, expiresAt time.Time, subject string) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings{"hermes-logos"},
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		Issuer:    "https://issuer.example/",
		NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		Subject:   subject,
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatalf("SignedString(): %v", err)
	}
	return raw
}
