package config

import (
	"log/slog"
	"os"
)

type Config struct {
	Port            string
	OpenRouterKey   string
	Model           string
	OpenRouterURL   string
	CorsOrigins     []string
	LogLevel        slog.Level
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8081"),
		OpenRouterKey: mustEnv("OPENROUTER_API_KEY"),
		Model:         getEnv("MODEL", "deepseek/deepseek-chat-v3-0324"),
		OpenRouterURL: getEnv("OPENROUTER_URL", "https://openrouter.ai/api/v1"),
		CorsOrigins:   []string{getEnv("CORS_ORIGINS", "*")},
		LogLevel:      parseLogLevel(getEnv("LOG_LEVEL", "info")),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var missing: " + key)
	}
	return v
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
