package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"hermes-logos/internal/bootstrap"
	"hermes-logos/internal/config"
	httpHandlers "hermes-logos/internal/handlers/http"
	"hermes-logos/internal/prompts"
	"hermes-logos/internal/tools"
)

func main() {
	cfg := config.Load()
	bootstrap.SetupLogger(cfg.LogLevel)

	logPromptConfig(cfg.PromptsBaseDir)

	loadedTools := tools.MustLoad(cfg.CommandsFile)

	slog.Info("loaded", "tools_count", len(loadedTools))

	chatHandler := httpHandlers.NewChatHandler(httpHandlers.Config{
		APIKey:          cfg.APIKey,
		BaseURL:         cfg.BaseURL,
		Model:           cfg.Model,
		PromptsBaseDir:  cfg.PromptsBaseDir,
		Tools:           loadedTools,
		Verbose:         cfg.Verbose,
		GPTVerbosity:    cfg.GPTVerbosity,
		ReasoningEffort: cfg.ReasoningEffort,
		Pricing: httpHandlers.Pricing{
			InputCentsPerMillion:       cfg.InputTokenCost,
			OutputCentsPerMillion:      cfg.OutputTokenCost,
			CachedInputCentsPerMillion: cfg.CachedInputTokenCost,
		},
	})

	r := chi.NewRouter()
	httpHandlers.SetupRoutes(r, chatHandler, cfg.CorsOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "port", cfg.Port, "model", cfg.Model)
	log.Fatal(http.ListenAndServe(addr, r))
}

func logPromptConfig(baseDir string) {
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		slog.Warn("failed to resolve prompt path", "error", err, "path", baseDir)
		return
	}

	dirs, err := prompts.ListDirectories(baseDir)
	if err != nil {
		slog.Warn("failed to list prompt directories", "error", err, "path", absPath)
		return
	}

	slog.Info("prompts", "path", absPath, "available_dirs", dirs)
}
