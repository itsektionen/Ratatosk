package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/itsektionen/ratatosk/internal/app"
	"github.com/itsektionen/ratatosk/internal/auth"
	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/db"
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
		handler = server.Root(cfg, newApp(cfg), server.Sites(cfg))
	case "serve-app":
		handler = newApp(cfg)
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

func newApp(cfg config.Config) http.Handler {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := db.Open(ctx, cfg.DataDir)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	sites.PrepareDir(cfg.SitesDir)
	store := sites.NewStore(database, cfg)
	if err := store.Adopt(ctx); err != nil {
		slog.Error("adopt existing sites", "err", err)
		os.Exit(1)
	}
	a, err := auth.New(ctx, cfg, database)
	if err != nil {
		slog.Error("set up auth", "err", err)
		os.Exit(1)
	}
	return app.Handler(cfg, a, store)
}
