package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func SetupRoutes(r *chi.Mux, chatHandler http.HandlerFunc, embeddingsHandler http.HandlerFunc, authMiddleware func(http.Handler) http.Handler, corsOrigins, requestHeaders []string) {
	r.Use(middleware.Recoverer)
	r.Use(RequestContext(requestHeaders))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   allowedHeaders(requestHeaders),
		AllowCredentials: true,
	}))

	r.Group(func(agents chi.Router) {
		agents.Use(authMiddleware)
		agents.Post("/embeddings", embeddingsHandler)
		agents.Post("/*", chatHandler)
	})
}

func allowedHeaders(requestHeaders []string) []string {
	headers := make([]string, 0, len(requestHeaders)+2)
	headers = append(headers, "Content-Type", "Authorization")
	return append(headers, requestHeaders...)
}
