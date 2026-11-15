package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/itsektionen/ratatosk/web"
)

func rootHandler(cfg Config, app, sites http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hostname(r) == cfg.BaseDomain {
			app.ServeHTTP(w, r)
			return
		}
		sites.ServeHTTP(w, r)
	})
}

// Returns the request host, lowercased and without port
func hostname(r *http.Request) string {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	// Fun fact: as far as DNS cares, example.com == example.com.
	return strings.ToLower(strings.TrimSuffix(host, "."))
}

func appHandler(cfg Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer)

	r.Route("/api", func(api chi.Router) {
		api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "ok")
		})
		api.Post("/upload", uploadHandler(cfg))
	})

	r.Handle("/*", web.Handler())

	return r
}
