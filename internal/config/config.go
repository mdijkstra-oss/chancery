package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/mdijkstra-oss/chancery/internal/auth"
	"github.com/mdijkstra-oss/chancery/internal/responses"
)

type Config struct {
	Auth            auth.Config
	Backend         responses.Config
	Port            string
	CorsOrigins     []string
	LogLevel        slog.Level
	Environment     string
	RequestHeaders  []string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	backend, err := LoadBackend()
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "60s"))
	if err != nil {
		return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT: %w", err)
	}
	logLevel, err := parseLogLevel(getEnv("LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		Auth: auth.Config{
			JWKSURL:       getEnv("AUTH_JWT_JWKS_URL", ""),
			PublicKeyFile: getEnv("AUTH_JWT_PUBLIC_KEY_FILE", ""),
			Issuer:        getEnv("AUTH_JWT_ISSUER", ""),
			Audience:      getEnv("AUTH_JWT_AUDIENCE", ""),
			Algorithms:    parseList(getEnv("AUTH_JWT_ALGORITHMS", "")),
		},
		Backend:         backend,
		Port:            ListenPort(),
		CorsOrigins:     parseList(getEnv("CORS_ORIGINS", "")),
		LogLevel:        logLevel,
		Environment:     getEnv("ENV", "development"),
		RequestHeaders:  requestHeaders(),
		ShutdownTimeout: shutdownTimeout,
	}
	if err := cfg.Auth.Validate(); err != nil {
		return Config{}, err
	}
	if err := validateRequestHeaders(cfg.RequestHeaders); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LoadBackend reads the two variables that reach the backend. The base URL has no
// default: silently assuming localhost fails as a refused connection deep in a
// request rather than at boot, where the mistake is.
func LoadBackend() (responses.Config, error) {
	cfg := responses.Config{
		BaseURL:   os.Getenv("RESPONSES_BASE_URL"),
		AuthToken: os.Getenv("RESPONSES_AUTH_TOKEN"),
	}
	if cfg.BaseURL == "" {
		return responses.Config{}, errors.New("RESPONSES_BASE_URL is required")
	}
	return cfg, nil
}

// ListenPort is the port the server binds, read on its own so that a command asking
// where this process would be listening does not have to load a whole configuration
// it has no other use for.
func ListenPort() string {
	return getEnv("PORT", "8081")
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

// A misspelled level is the operator's mistake, not the program's, so it is reported
// the way every other malformed variable here is.
func parseLogLevel(value string) (slog.Level, error) {
	switch value {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL %q must be debug, info, warn, or error", value)
	}
}
