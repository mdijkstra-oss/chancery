package config

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

var configEnvKeys = []string{
	"AUTH_JWT_JWKS_URL",
	"AUTH_JWT_PUBLIC_KEY_FILE",
	"AUTH_JWT_ISSUER",
	"AUTH_JWT_AUDIENCE",
	"AUTH_JWT_ALGORITHMS",
	"LOG_REQUEST_HEADERS",
	"PORT",
	"CORS_ORIGINS",
	"LOG_LEVEL",
	"ENV",
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantEnabled bool
		wantHeaders []string
		wantError   string
	}{
		{name: "disabled defaults", wantHeaders: []string{"X-Session-ID", "X-Project-ID"}},
		{
			name: "JWKS enabled",
			env: map[string]string{
				"AUTH_JWT_JWKS_URL":   "https://issuer.example/jwks.json",
				"AUTH_JWT_ISSUER":     "https://issuer.example/",
				"AUTH_JWT_AUDIENCE":   "hermes-logos",
				"AUTH_JWT_ALGORITHMS": "RS256, PS256,RS256",
				"LOG_REQUEST_HEADERS": "X-Tenant-ID,X-Trace-ID",
			},
			wantEnabled: true,
			wantHeaders: []string{"X-Tenant-ID", "X-Trace-ID"},
		},
		{
			name: "static key enabled",
			env: map[string]string{
				"AUTH_JWT_PUBLIC_KEY_FILE": "/run/keys/jwt.pem",
				"AUTH_JWT_ISSUER":          "issuer",
				"AUTH_JWT_AUDIENCE":        "hermes-logos",
				"AUTH_JWT_ALGORITHMS":      "EdDSA",
			},
			wantEnabled: true,
			wantHeaders: []string{"X-Session-ID", "X-Project-ID"},
		},
		{
			name: "conflicting key sources",
			env: map[string]string{
				"AUTH_JWT_JWKS_URL":        "https://issuer.example/jwks.json",
				"AUTH_JWT_PUBLIC_KEY_FILE": "/run/keys/jwt.pem",
			},
			wantError: "mutually exclusive",
		},
		{
			name: "missing issuer",
			env: map[string]string{
				"AUTH_JWT_PUBLIC_KEY_FILE": "/run/keys/jwt.pem",
				"AUTH_JWT_AUDIENCE":        "hermes-logos",
				"AUTH_JWT_ALGORITHMS":      "RS256",
			},
			wantError: "AUTH_JWT_ISSUER",
		},
		{
			name: "missing audience",
			env: map[string]string{
				"AUTH_JWT_PUBLIC_KEY_FILE": "/run/keys/jwt.pem",
				"AUTH_JWT_ISSUER":          "issuer",
				"AUTH_JWT_ALGORITHMS":      "RS256",
			},
			wantError: "AUTH_JWT_AUDIENCE",
		},
		{
			name: "missing algorithms",
			env: map[string]string{
				"AUTH_JWT_PUBLIC_KEY_FILE": "/run/keys/jwt.pem",
				"AUTH_JWT_ISSUER":          "issuer",
				"AUTH_JWT_AUDIENCE":        "hermes-logos",
			},
			wantError: "AUTH_JWT_ALGORITHMS",
		},
		{
			name: "insecure JWKS URL",
			env: map[string]string{
				"AUTH_JWT_JWKS_URL":   "http://issuer.example/jwks.json",
				"AUTH_JWT_ISSUER":     "issuer",
				"AUTH_JWT_AUDIENCE":   "hermes-logos",
				"AUTH_JWT_ALGORITHMS": "RS256",
			},
			wantError: "HTTPS URL",
		},
		{
			name: "symmetric algorithm rejected",
			env: map[string]string{
				"AUTH_JWT_PUBLIC_KEY_FILE": "/run/keys/jwt.pem",
				"AUTH_JWT_ISSUER":          "issuer",
				"AUTH_JWT_AUDIENCE":        "hermes-logos",
				"AUTH_JWT_ALGORITHMS":      "HS256",
			},
			wantError: "unsupported JWT algorithm",
		},
		{
			name: "credential header rejected",
			env: map[string]string{
				"LOG_REQUEST_HEADERS": "X-API-Key",
			},
			wantError: "credential header",
		},
		{
			name: "non X header rejected",
			env: map[string]string{
				"LOG_REQUEST_HEADERS": "Traceparent",
			},
			wantError: "invalid header",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setConfigEnv(t, test.env)
			got, err := Load()
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("Load() error = %v, want containing %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load(): %v", err)
			}
			if got.Auth.Enabled() != test.wantEnabled {
				t.Errorf("Auth.Enabled() = %v, want %v", got.Auth.Enabled(), test.wantEnabled)
			}
			if !reflect.DeepEqual(got.RequestHeaders, test.wantHeaders) {
				t.Errorf("RequestHeaders = %#v, want %#v", got.RequestHeaders, test.wantHeaders)
			}
		})
	}
}

func setConfigEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for _, key := range configEnvKeys {
		previous, existed := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Unsetenv(%s): %v", key, err)
		}
		t.Cleanup(restoreEnv(key, previous, existed))
	}
	for key, value := range values {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("Setenv(%s): %v", key, err)
		}
	}
}

func restoreEnv(key, value string, existed bool) func() {
	return func() {
		if existed {
			os.Setenv(key, value)
			return
		}
		os.Unsetenv(key)
	}
}
