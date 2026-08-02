package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mdijkstra-oss/chancery/internal/auth"
	"github.com/mdijkstra-oss/chancery/internal/bootstrap"
	"github.com/mdijkstra-oss/chancery/internal/config"
	httpHandlers "github.com/mdijkstra-oss/chancery/internal/handlers/http"
	"github.com/mdijkstra-oss/chancery/internal/prompts"
	"github.com/mdijkstra-oss/chancery/internal/quota"
	"github.com/mdijkstra-oss/chancery/internal/ratelimit"

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
	srv := &http.Server{
		Addr:              ":" + runtimeConfig.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("serve: %w", err)
			return
		}
		serveErr <- nil
	}()

	slog.Info("server starting", "component", "startup", slog.Group("data", slog.String("port", runtimeConfig.Port)))

	select {
	case err := <-serveErr:
		return err
	case <-shutdownCtx.Done():
		slog.Info("server shutting down", "component", "startup", slog.Group("data", slog.Duration("grace", runtimeConfig.ShutdownTimeout)))
		graceCtx, cancel := context.WithTimeout(context.Background(), runtimeConfig.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(graceCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	}
}
