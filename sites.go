package main

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var dnsRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func sitesHandler(cfg Config) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name, ok := strings.CutSuffix(hostname(r), "."+cfg.BaseDomain)
		if !ok || !dnsRe.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		// The root keeps pointing at the same version of the site even if a
		// deploy swaps it out mid-request
		root, err := os.OpenRoot(filepath.Join(cfg.SitesDir, name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer root.Close()
		serveSite(w, r, root)
	})
	return chi.Chain(middleware.RequestID, middleware.Logger, middleware.Recoverer).Handler(h)
}

func serveSite(w http.ResponseWriter, r *http.Request, root *os.Root) {
	// Always revalidate, so a new deploy shows up immediately
	w.Header().Set("Cache-Control", "no-cache")

	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "."
	}
	// Prevent the 404 page and directory listing from Go's file server
	fi, err := root.Stat(name)
	if err == nil && fi.IsDir() {
		fi, err = root.Stat(path.Join(name, "index.html"))
	}
	if err != nil || !fi.Mode().IsRegular() {
		serveNotFound(w, root)
		return
	}

	http.FileServerFS(root.FS()).ServeHTTP(w, r)
}

func serveNotFound(w http.ResponseWriter, root *os.Root) {
	f, err := root.Open("404.html")
	if err != nil {
		// Fallback 404 if the site has none
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	io.Copy(w, f)
}
