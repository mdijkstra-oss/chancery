package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

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

	systemPrompt := prompts.MustLoad(cfg.SystemPromptPath)
	loadedTools := tools.MustLoad(cfg.CommandsFile)

	slog.Info("loaded",
		"system_prompt_len", len(systemPrompt),
		"tools_count", len(loadedTools),
	)

	chatHandler := httpHandlers.NewChatHandler(httpHandlers.Config{
		APIKey:           cfg.OpenRouterKey,
		BaseURL:          cfg.OpenRouterURL,
		Model:            cfg.Model,
		Provider:         cfg.Provider,
		SystemPrompt:     systemPrompt,
		Tools:            loadedTools,
		Verbose:          cfg.Verbose,
		IncludeReasoning: cfg.IncludeReasoning,
		CacheInterval:    cfg.CacheInterval,
		MaxTokenWindow:   cfg.MaxTokenWindow,
	})

	r := chi.NewRouter()
	httpHandlers.SetupRoutes(r, chatHandler, cfg.CorsOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "port", cfg.Port, "model", cfg.Model)
	log.Fatal(http.ListenAndServe(addr, r))
}
