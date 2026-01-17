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
)

func main() {
	cfg := config.Load()
	bootstrap.SetupLogger(cfg.LogLevel)

	if err := config.ValidateEndpoints(); err != nil {
		log.Fatal(err)
	}

	logEndpoints()

	chatHandler := httpHandlers.NewChatHandler(httpHandlers.Config{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Verbose: cfg.Verbose,
		Pricing: httpHandlers.Pricing{
			InputCentsPerMillion:       cfg.InputTokenCost,
			OutputCentsPerMillion:      cfg.OutputTokenCost,
			CachedInputCentsPerMillion: cfg.CachedInputTokenCost,
		},
	})

	r := chi.NewRouter()
	httpHandlers.SetupRoutes(r, chatHandler, cfg.CorsOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "port", cfg.Port)
	log.Fatal(http.ListenAndServe(addr, r))
}

func logEndpoints() {
	endpoints := make([]string, 0, len(config.Endpoints))
	for name := range config.Endpoints {
		endpoints = append(endpoints, name)
	}
	slog.Info("endpoints", "available", endpoints)
}
