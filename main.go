package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/itsektionen/ratatosk/internal/app"
	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/server"
	"github.com/itsektionen/ratatosk/internal/sites"
)

func main() {
	cfg := config.Load()

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var handler http.Handler
	switch cmd {
	case "serve":
		sites.PrepareDir(cfg.SitesDir)
		handler = server.Root(cfg, app.Handler(cfg), server.Sites(cfg))
	case "serve-app":
		sites.PrepareDir(cfg.SitesDir)
		handler = app.Handler(cfg)
	case "serve-sites":
		handler = server.Sites(cfg)
	default:
		slog.Error("unknown command", "cmd", cmd, "valid", "serve, serve-app, serve-sites")
		os.Exit(1)
	}

	slog.Info("listening", "cmd", cmd, "addr", cfg.Addr)
	if err := server.Run(cfg.Addr, handler); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
