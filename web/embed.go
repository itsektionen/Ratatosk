package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:build
var webFS embed.FS

func siteHandler() http.Handler {
}
