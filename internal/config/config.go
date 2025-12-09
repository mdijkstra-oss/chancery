package config

import (
	"hermes-logos/internal/utils"
	"log/slog"
	"os"
)

type Config struct {
	Port             string
	OpenRouterKey    string
	Model            string
	Provider         string
	OpenRouterURL    string
	CorsOrigins      []string
	LogLevel         slog.Level
	CommandsFile     string
	SystemPromptPath string
	Debug            bool
	IncludeReasoning bool
}

func Load() Config {
	return Config{
		Port:             getEnv("PORT", "8081"),
		OpenRouterKey:    "REDACTED-OPENROUTER-API-KEY",
		// Caching: DeepSeek/OpenAI/Gemini 2.5 = automatic, Anthropic/Gemini = needs cache_control in messages.
		// Not all providers support caching. Use PROVIDER to pin to a specific provider for consistent cache hits.
		// Privacy settings affect provider availability: https://openrouter.ai/settings/privacy
		// Self-hosting models like DeepSeek gives full control over caching and data privacy.
		Model:    getEnv("MODEL", "deepseek/deepseek-v3.2"),
		Provider: os.Getenv("PROVIDER"),
		OpenRouterURL:    getEnv("OPENROUTER_URL", "https://openrouter.ai/api/v1"),
		CorsOrigins:      []string{getEnv("CORS_ORIGINS", "*")},
		LogLevel:         parseLogLevel(getEnv("LOG_LEVEL", "info")),
		CommandsFile:     "/home/hermes/hermes-mcp/tools.json",
		SystemPromptPath: "/home/hermes/hermes-mcp/prompts/",
		Debug:            os.Getenv("DEBUG") != "",
		IncludeReasoning: utils.IsTruthy(os.Getenv("INCLUDE_REASONING")),
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
