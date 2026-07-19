package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/matthijn/hermes-logos/internal/auth"
	"github.com/matthijn/hermes-logos/internal/bootstrap"
	"github.com/matthijn/hermes-logos/internal/config"
	httpHandlers "github.com/matthijn/hermes-logos/internal/handlers/http"
	"github.com/matthijn/hermes-logos/internal/prompts"
	"github.com/matthijn/hermes-logos/internal/quota"
	"github.com/matthijn/hermes-logos/internal/ratelimit"

	"github.com/go-chi/chi/v5"
)

func Run(ctx context.Context, registry prompts.Registry) error {
	resolvedRegistry, err := registry.WithAPIKeys(os.Getenv)
	if err != nil {
		return err
	}
	embeddings, err := resolvedRegistry.ResolveAgent("embeddings")
	if err != nil {
		return fmt.Errorf("embeddings agent is required for serve")
	}

	runtimeConfig, err := config.Load()
	if err != nil {
		return err
	}
	bootstrap.SetupLogger(runtimeConfig.LogLevel, runtimeConfig.Environment)
	validator, err := auth.NewValidator(ctx, runtimeConfig.Auth)
	if err != nil {
		return err
	}
	defer validator.Close()
	if !validator.Enabled() {
		slog.Warn("auth disabled — all requests accepted")
	}
	slog.Info("config loaded", "component", "startup", slog.Group("data", slog.Int("agents", len(resolvedRegistry.Agents))))
	limiter := ratelimit.NewLimiter()
	quotaClient := quota.NewClient(runtimeConfig.Quota)
	chatHandler := httpHandlers.NewChatHandler(resolvedRegistry, limiter, quotaClient)
	embeddingsHandler := httpHandlers.NewEmbeddingsHandler(embeddings.Config, limiter, quotaClient)
	router := chi.NewRouter()
	httpHandlers.SetupRoutes(router, chatHandler, embeddingsHandler, httpHandlers.JWTAuthentication(validator), runtimeConfig.CorsOrigins, runtimeConfig.RequestHeaders)
	address := ":" + runtimeConfig.Port
	slog.Info("server starting", "component", "startup", slog.Group("data", slog.String("port", runtimeConfig.Port)))
	if err := http.ListenAndServe(address, router); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
