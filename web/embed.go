package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:build
var webFS embed.FS

// Serves the prerendered embedded site
func Handler() http.Handler {
	build, err := fs.Sub(webFS, "build")
	if err != nil {
		panic(err)
	}
	return http.FileServerFS(build)
}
