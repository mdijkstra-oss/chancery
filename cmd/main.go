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

	registry := prompts.CompileRegistry(prompts.PromptsDir)
	slog.Info("prompts compiled", "component", "startup", slog.Group("data", slog.Int("agents", len(registry.Agents))))

	chatHandler := httpHandlers.NewChatHandler(httpHandlers.Config{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Inspect: cfg.Inspect,
	}, registry)

	approachesHandler := httpHandlers.NewApproachesHandler(registry.Approaches)

	embCfg := registry.Configs["embeddings"]
	embeddingsHandler := httpHandlers.NewEmbeddingsHandler(httpHandlers.Config{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	}, httpHandlers.EmbeddingConfig{
		Model:      embCfg.Model,
		Dimensions: embCfg.Dimensions,
	})

	r := chi.NewRouter()
	httpHandlers.SetupRoutes(r, chatHandler, approachesHandler, embeddingsHandler, cfg.CorsOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "component", "startup", slog.Group("data", slog.String("port", cfg.Port)))
	log.Fatal(http.ListenAndServe(addr, r))
}
