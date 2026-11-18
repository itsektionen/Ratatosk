package server

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/sites"
)

func Sites(cfg config.Config) http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sites are public files. Previews run sandboxed, with no origin of their own, and
		// need this to load their own files with fetch(), fonts and p5's loadImage()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name, ok := strings.CutSuffix(Hostname(r), "."+cfg.BaseDomain)
		if !ok || !sites.ValidName(name) {
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
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	io.Copy(w, f)
}
