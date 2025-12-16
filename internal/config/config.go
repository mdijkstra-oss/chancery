package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	Port                 string
	APIKey               string
	Model                string
	BaseURL              string
	CorsOrigins          []string
	LogLevel             slog.Level
	CommandsFile         string
	PromptsBaseDir       string
	Verbose              bool
	GPTVerbosity         string
	ReasoningEffort      string
	InputTokenCost       float64
	OutputTokenCost      float64
	CachedInputTokenCost float64
}

func Load() Config {
	return Config{
		Port:                 getEnv("PORT", "8081"),
		APIKey:               getEnv("API_KEY", ""),
		Model:                getEnv("MODEL", "gpt-5.2"),
		BaseURL:              getEnv("BASE_URL", "https://api.openai.com/v1"),
		CorsOrigins:          []string{getEnv("CORS_ORIGINS", "*")},
		LogLevel:             parseLogLevel(getEnv("LOG_LEVEL", "info")),
		CommandsFile:         getEnv("COMMANDS_FILE", "/home/hermes/hermes-mcp/tools.json"),
		PromptsBaseDir:       getEnv("PROMPTS_BASE_DIR", "/home/hermes/hermes-mcp/prompts/"),
		Verbose:              os.Getenv("VERBOSE") != "",
		GPTVerbosity:         os.Getenv("GPT_VERBOSITY"),
		ReasoningEffort:      os.Getenv("REASONING_EFFORT"),
		InputTokenCost:       parseFloat(getEnv("INPUT_TOKEN_COST", "125")),
		OutputTokenCost:      parseFloat(getEnv("OUTPUT_TOKEN_COST", "1000")),
		CachedInputTokenCost: parseFloat(getEnv("CACHED_INPUT_TOKEN_COST", "12.5")),
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

func parseInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		panic("invalid integer: " + s)
	}
	return v
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic("invalid float: " + s)
	}
	return v
}
