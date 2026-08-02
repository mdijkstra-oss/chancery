package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidator(t *testing.T) {
	key := generateRSAKey(t)
	wrongKey := generateRSAKey(t)
	publicKeyFile := writePublicKey(t, &key.PublicKey)
	cfg := Config{
		PublicKeyFile: publicKeyFile,
		Issuer:        "https://issuer.example/",
		Audience:      "chancery",
		Algorithms:    []string{"RS256"},
	}
	validator, err := NewValidator(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewValidator(): %v", err)
	}
	t.Cleanup(validator.Close)
	now := time.Now()
	tests := []struct {
		name      string
		key       *rsa.PrivateKey
		claims    jwt.RegisteredClaims
		wantUser  string
		wantError bool
	}{
		{
			name:     "valid token",
			key:      key,
			claims:   testClaims(now.Add(time.Hour), now.Add(-time.Second), "user-123"),
			wantUser: "user-123",
		},
		{
			name:      "expired token",
			key:       key,
			claims:    testClaims(now.Add(-time.Minute), now.Add(-time.Hour), "user-123"),
			wantError: true,
		},
		{
			name:      "bad signature",
			key:       wrongKey,
			claims:    testClaims(now.Add(time.Hour), now.Add(-time.Second), "user-123"),
			wantError: true,
		},
		{
			name:      "not before in future",
			key:       key,
			claims:    testClaims(now.Add(time.Hour), now.Add(time.Minute), "user-123"),
			wantError: true,
		},
		{
			name:      "missing subject",
			key:       key,
			claims:    testClaims(now.Add(time.Hour), now.Add(-time.Second), ""),
			wantError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := signToken(t, test.key, test.claims)
			got, validateErr := validator.Validate(context.Background(), raw)
			if test.wantError {
				if validateErr == nil {
					t.Fatal("Validate() error = nil")
				}
				return
			}
			if validateErr != nil {
				t.Fatalf("Validate(): %v", validateErr)
			}
			if got != test.wantUser {
				t.Errorf("Validate() user = %q, want %q", got, test.wantUser)
			}
		})
	}
}

func TestJWKSRefreshesUnknownKID(t *testing.T) {
	first := generateRSAKey(t)
	second := generateRSAKey(t)
	var requests atomic.Int32
	server := httptest.NewTLSServer(rotatingJWKSHandler(&first.PublicKey, &second.PublicKey, &requests))
	t.Cleanup(server.Close)
	validator, err := newValidator(context.Background(), Config{
		JWKSURL:    server.URL,
		Issuer:     "https://issuer.example/",
		Audience:   "chancery",
		Algorithms: []string{"RS256"},
	}, server.Client())
	if err != nil {
		t.Fatalf("NewValidator(): %v", err)
	}
	t.Cleanup(validator.Close)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, testClaims(time.Now().Add(time.Hour), time.Now().Add(-time.Minute), "user-123"))
	token.Header["kid"] = "second"
	raw, err := token.SignedString(second)
	if err != nil {
		t.Fatalf("SignedString(): %v", err)
	}
	user, err := validator.Validate(context.Background(), raw)
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
	if user != "user-123" {
		t.Errorf("user = %q, want user-123", user)
	}
	if requests.Load() < 2 {
		t.Errorf("JWKS requests = %d, want at least 2", requests.Load())
	}
}

func TestDisabledValidator(t *testing.T) {
	validator, err := NewValidator(context.Background(), Config{})
	if err != nil {
		t.Fatalf("NewValidator(): %v", err)
	}
	if validator.Enabled() {
		t.Error("Enabled() = true, want false")
	}
	user, err := validator.Validate(context.Background(), "not-a-token")
	if err != nil {
		t.Fatalf("Validate(): %v", err)
	}
	if user != "" {
		t.Errorf("Validate() user = %q, want empty", user)
	}
}

type testJWK struct {
	Algorithm string `json:"alg"`
	Exponent  string `json:"e"`
	KeyID     string `json:"kid"`
	KeyType   string `json:"kty"`
	Modulus   string `json:"n"`
	Use       string `json:"use"`
}

type testJWKS struct {
	Keys []testJWK `json:"keys"`
}

func rotatingJWKSHandler(first, second *rsa.PublicKey, requests *atomic.Int32) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		request := requests.Add(1)
		key := first
		keyID := "first"
		if request > 1 {
			key = second
			keyID = "second"
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(testJWKS{Keys: []testJWK{jwkFromRSA(key, keyID)}}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func jwkFromRSA(key *rsa.PublicKey, keyID string) testJWK {
	return testJWK{
		Algorithm: "RS256",
		Exponent:  base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		KeyID:     keyID,
		KeyType:   "RSA",
		Modulus:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		Use:       "sig",
	}
}

func testClaims(expiresAt, notBefore time.Time, subject string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Audience:  jwt.ClaimStrings{"chancery"},
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		Issuer:    "https://issuer.example/",
		NotBefore: jwt.NewNumericDate(notBefore),
		Subject:   subject,
	}
}

func signToken(t *testing.T, key *rsa.PrivateKey, claims jwt.RegisteredClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString(): %v", err)
	}
	return raw
}

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey(): %v", err)
	}
	return key
}

func writePublicKey(t *testing.T, key *rsa.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey(): %v", err)
	}
	path := filepath.Join(t.TempDir(), "public.pem")
	data := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}
	return path
}
