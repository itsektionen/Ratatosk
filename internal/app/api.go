package app

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/itsektionen/ratatosk/internal/auth"
	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/sites"
	"github.com/itsektionen/ratatosk/web"
)

func Handler(cfg config.Config, a *auth.Auth, store *sites.Store) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer, denyFraming)

	r.Route("/api", func(api chi.Router) {
		api.Use(a.Middleware, sameOriginOnly(cfg))
		api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "ok")
		})
		api.Mount("/auth", a.Handler())
		api.Get("/boot", bootHandler(cfg, a, store))
		api.Get("/sites", listSites(store))
		api.With(auth.RequireUser).Group(func(r chi.Router) {
			r.Post("/upload", uploadHandler(cfg, store))
			r.Delete("/sites/{name}", deleteSite(store))
		})
	})

	r.With(a.Middleware).Handle("/*", web.Handler(bootJSON(cfg, a, store)))

	return r
}
