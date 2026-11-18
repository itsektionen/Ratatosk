package web

import (
	"bytes"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
)

//go:embed all:build
var webFS embed.FS

// Serves the built web app. boot returns JSON that is embedded into every HTML
// page, so the page starts with its data instead of fetching it
func Handler(boot func(*http.Request) ([]byte, error)) http.Handler {
	build, err := fs.Sub(webFS, "build")
	if err != nil {
		panic(err)
	}
	files := http.FileServerFS(build)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "."
		}

		// A directory is only served if it has an index.html. This also stops
		// the file server from listing directories like _app/
		target := name
		if fi, err := fs.Stat(build, name); err == nil && fi.IsDir() {
			target = path.Join(name, "index.html")
		} else if err != nil {
			// Prerendered routes are built as <route>.html
			target = name + ".html"
		}
		if fi, err := fs.Stat(build, target); err == nil && fi.Mode().IsRegular() {
			switch {
			case path.Ext(target) == ".html" && r.URL.Path != "/"+target:
				serveHTML(w, r, build, target, http.StatusOK, boot)
			case strings.HasPrefix(name, "_app/immutable/"):
				// Everything here has a hash in its name
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				files.ServeHTTP(w, r)
			default:
				w.Header().Set("Cache-Control", "no-cache")
				files.ServeHTTP(w, r)
			}
			return
		}

		serveHTML(w, r, build, "404.html", http.StatusNotFound, boot)
	})
}

// serveHTML serves a page with the boot data in its <head>. It's different per
// user, so it must never be cached
func serveHTML(w http.ResponseWriter, r *http.Request, build fs.FS, file string, status int, boot func(*http.Request) ([]byte, error)) {
	page, err := fs.ReadFile(build, file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if data, err := boot(r); err != nil {
		// The page falls back to fetching /api/boot
		slog.Error("boot data", "err", err)
	} else {
		tag := append(append([]byte(`<script type="application/json" id="boot">`), data...), "</script>"...)
		page = bytes.Replace(page, []byte("</head>"), append(tag, "</head>"...), 1)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		w.Write(page)
	}
}
