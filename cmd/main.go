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
)

func main() {
	cfg := config.Load()
	bootstrap.SetupLogger(cfg.LogLevel)

	promptResult := prompts.MustLoad(cfg.CommandsFile)
	totalTokens := promptResult.SystemTokens + promptResult.CommandTokens
	slog.Info("prompts loaded",
		"system_tokens", promptResult.SystemTokens,
		"command_tokens", promptResult.CommandTokens,
		"total_tokens", totalTokens,
	)

	streamHandler := httpHandlers.NewStreamHandler(cfg.OpenRouterKey, cfg.OpenRouterURL, cfg.Model, promptResult.Combined)

	r := chi.NewRouter()
	httpHandlers.SetupRoutes(r, streamHandler, cfg.CorsOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "port", cfg.Port, "model", cfg.Model)
	log.Fatal(http.ListenAndServe(addr, r))
}
