package config

import (
	"log/slog"
	"os"
)

type Config struct {
	Port        string
	CorsOrigins []string
	LogLevel    slog.Level
	Environment string
}

func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8081"),
		CorsOrigins: []string{getEnv("CORS_ORIGINS", "*")},
		LogLevel:    parseLogLevel(getEnv("LOG_LEVEL", "info")),
		Environment: getEnv("ENV", "development"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
