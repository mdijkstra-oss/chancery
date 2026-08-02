package config

import (
	"fmt"
	"log/slog"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/mdijkstra-oss/chancery/internal/auth"
	"github.com/mdijkstra-oss/chancery/internal/quota"
)

type Config struct {
	Auth            auth.Config
	Quota           quota.Config
	Port            string
	CorsOrigins     []string
	LogLevel        slog.Level
	Environment     string
	RequestHeaders  []string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	quotaTimeout, err := time.ParseDuration(getEnv("QUOTA_TIMEOUT", "2s"))
	if err != nil {
		return Config{}, fmt.Errorf("QUOTA_TIMEOUT: %w", err)
	}
	shutdownTimeout, err := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "60s"))
	if err != nil {
		return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err)
	}
	cfg := Config{
		Auth: auth.Config{
			JWKSURL:       getEnv("AUTH_JWT_JWKS_URL", ""),
			PublicKeyFile: getEnv("AUTH_JWT_PUBLIC_KEY_FILE", ""),
			Issuer:        getEnv("AUTH_JWT_ISSUER", ""),
			Audience:      getEnv("AUTH_JWT_AUDIENCE", ""),
			Algorithms:    parseList(getEnv("AUTH_JWT_ALGORITHMS", "")),
		},
		Quota: quota.Config{
			ReserveURL: getEnv("QUOTA_RESERVE_URL", ""),
			SettleURL:  getEnv("QUOTA_SETTLE_URL", ""),
			AuthToken:  getEnv("QUOTA_AUTH_TOKEN", ""),
			Timeout:    quotaTimeout,
		},
		Port:            getEnv("PORT", "8081"),
		CorsOrigins:     parseList(getEnv("CORS_ORIGINS", "")),
		LogLevel:        parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Environment:     getEnv("ENV", "development"),
		RequestHeaders:  requestHeaders(),
		ShutdownTimeout: shutdownTimeout,
	}
	if err := cfg.Auth.Validate(); err != nil {
		return Config{}, err
	}
	if err := cfg.Quota.Validate(); err != nil {
		return Config{}, err
	}
	if err := validateRequestHeaders(cfg.RequestHeaders); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requestHeaders() []string {
	value, exists := os.LookupEnv("LOG_REQUEST_HEADERS")
	if !exists {
		return []string{"X-Session-ID", "X-Project-ID"}
	}
	return parseList(value)
}

func parseList(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		values = append(values, value)
	}
	return values
}

func validateRequestHeaders(headers []string) error {
	if len(headers) > 16 {
		return fmt.Errorf("LOG_REQUEST_HEADERS contains more than 16 headers")
	}
	for _, header := range headers {
		canonical := textproto.CanonicalMIMEHeaderKey(header)
		if canonical == "" || !strings.HasPrefix(canonical, "X-") {
			return fmt.Errorf("LOG_REQUEST_HEADERS contains invalid header %q", header)
		}
		if isCredentialHeader(canonical) {
			return fmt.Errorf("LOG_REQUEST_HEADERS contains credential header %q", header)
		}
	}
	return nil
}

func isCredentialHeader(header string) bool {
	normalized := strings.ToLower(header)
	forbiddenParts := []string{"access-token", "api-key", "auth", "credential", "password", "secret", "security-token"}
	for _, part := range forbiddenParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		panic("unknown log level: " + s)
	}
}
