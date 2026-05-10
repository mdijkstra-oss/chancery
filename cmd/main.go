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
	"hermes-logos/internal/ratelimit"
)

func main() {
	cfg := config.Load()
	bootstrap.SetupLogger(cfg.LogLevel, cfg.Environment)

	registry := prompts.CompileRegistry(prompts.PromptsDir)
	slog.Info("prompts compiled", "component", "startup", slog.Group("data", slog.Int("agents", len(registry.Agents))))

	limiter := ratelimit.NewLimiter()

	chatHandler := httpHandlers.NewChatHandler(registry, limiter)
	guidanceHandler := httpHandlers.NewGuidanceHandler(registry.Guidance)
	embeddingsHandler := httpHandlers.NewEmbeddingsHandler(mustEmbeddingsConfig(registry), limiter)

	r := chi.NewRouter()
	httpHandlers.SetupRoutes(r, chatHandler, guidanceHandler, embeddingsHandler, cfg.CorsOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "component", "startup", slog.Group("data", slog.String("port", cfg.Port)))
	log.Fatal(http.ListenAndServe(addr, r))
}

func mustEmbeddingsConfig(registry prompts.Registry) prompts.PromptConfig {
	cfg, ok := registry.Configs["embeddings"]
	if !ok {
		log.Fatal("config: embeddings agent not configured in agents.json")
	}
	return cfg
}
