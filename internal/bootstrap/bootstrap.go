package bootstrap

import (
	"log/slog"
	"os"

	"github.com/mdijkstra-oss/chancery/internal/logging"
)

func SetupLogger(level slog.Level, environment string) {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	contextHandler := logging.NewContextHandler(jsonHandler)
	logger := slog.New(contextHandler).With("environment", environment)
	slog.SetDefault(logger)
}
