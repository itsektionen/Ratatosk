package site

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web/build
var webFS embed.FS

func siteHandler() http.Handler {
	site, err := fs.Sub(webFS, "web/build")
	if err != nil {
		panic(err)
	}
}
