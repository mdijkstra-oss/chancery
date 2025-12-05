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

	streamHandler := httpHandlers.NewStreamHandler(cfg.OpenRouterKey, cfg.OpenRouterURL, cfg.Model)

	r := chi.NewRouter()
	httpHandlers.SetupRoutes(r, streamHandler, cfg.CorsOrigins)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "port", cfg.Port, "model", cfg.Model)
	log.Fatal(http.ListenAndServe(addr, r))
}
