package app

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/itsektionen/ratatosk/internal/auth"
	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/web"
)

func Handler(cfg config.Config, a *auth.Auth) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer, denyFraming)

	r.Route("/api", func(api chi.Router) {
		api.Use(a.Middleware, sameOriginOnly(cfg))
		api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "ok")
		})
		api.Mount("/auth", a.Handler())
		api.Post("/upload", uploadHandler(cfg))
	})

	r.Handle("/*", web.Handler())

	return r
}
