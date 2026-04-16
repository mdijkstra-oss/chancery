package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func SetupRoutes(r *chi.Mux, chatHandler http.HandlerFunc, approachesHandler http.HandlerFunc, embeddingsHandler http.HandlerFunc, corsOrigins []string) {
	r.Use(middleware.Recoverer)
	r.Use(RequestContext)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Session-ID", "X-Project-ID"},
		AllowCredentials: true,
	}))

	r.Get("/approaches", approachesHandler)
	r.Post("/embeddings", embeddingsHandler)
	r.Post("/*", chatHandler)
}
