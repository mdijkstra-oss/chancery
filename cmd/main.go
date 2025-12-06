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
	"hermes-logos/internal/lib/utils"
	"hermes-logos/internal/prompts"
)

func main() {
	cfg := config.Load()
	bootstrap.SetupLogger(cfg.LogLevel)

	systemPrompt := prompts.MustLoad(cfg.CommandsFile)
	promptTokens := utils.EstimateTokens(systemPrompt)
	slog.Info("system prompt loaded", "tokens", promptTokens, "chars", len(systemPrompt))

	streamHandler := httpHandlers.NewStreamHandler(cfg.OpenRouterKey, cfg.OpenRouterURL, cfg.Model, systemPrompt)

	r := chi.NewRouter()
	httpHandlers.SetupRoutes(r, streamHandler, cfg.CorsOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "port", cfg.Port, "model", cfg.Model)
	log.Fatal(http.ListenAndServe(addr, r))
}
