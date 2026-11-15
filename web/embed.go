package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:build
var webFS embed.FS

// Serves the built web app
func Handler() http.Handler {
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
		}
		if fi, err := fs.Stat(build, target); err == nil && fi.Mode().IsRegular() {
			if strings.HasPrefix(name, "_app/immutable/") {
				// Everything here has a hash in its name
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(w, r)
			return
		}

		notFound(w, r, build)
	})
}

func notFound(w http.ResponseWriter, r *http.Request, build fs.FS) {
	f, err := build.Open("404.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusNotFound)
	if r.Method != http.MethodHead {
		io.Copy(w, f)
	}
}
