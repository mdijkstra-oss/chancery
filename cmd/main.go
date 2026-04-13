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
	bootstrap.SetupLogger(cfg.LogLevel, cfg.Environment)

	registry := prompts.CompileRegistry(prompts.PromptsDir)
	slog.Info("prompts compiled", "component", "startup", slog.Group("data", slog.Int("agents", len(registry.Agents))))

	chatHandler := httpHandlers.NewChatHandler(cfg.Inspect, registry)
	approachesHandler := httpHandlers.NewApproachesHandler(registry.Approaches)

	r := chi.NewRouter()
	httpHandlers.SetupRoutes(r, chatHandler, approachesHandler, cfg.CorsOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "component", "startup", slog.Group("data", slog.String("port", cfg.Port)))
	log.Fatal(http.ListenAndServe(addr, r))
}
