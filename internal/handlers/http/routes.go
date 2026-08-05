package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

const maxRequestBodyBytes = 10 << 20

func SetupRoutes(r *chi.Mux, chatHandler http.HandlerFunc, authMiddleware func(http.Handler) http.Handler, corsOrigins, requestHeaders []string) {
	r.Use(middleware.Recoverer)
	r.Use(RequestContext(requestHeaders))
	r.Use(cors.Handler(corsOptions(corsOrigins)))

	r.Get("/health", healthHandler)
	r.Group(func(agents chi.Router) {
		agents.Use(authMiddleware)
		agents.Post("/*", chatHandler)
	})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

// The origin allowlist is the control. A header allowlist beside it decides only
// which headers a browser may send, which is a question about what this serves
// rather than who may reach it.
func corsOptions(corsOrigins []string) cors.Options {
	options := cors.Options{
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	}
	if len(corsOrigins) == 0 {
		options.AllowOriginFunc = denyAllOrigins
		return options
	}
	options.AllowedOrigins = corsOrigins
	return options
}

func denyAllOrigins(*http.Request, string) bool {
	return false
}
