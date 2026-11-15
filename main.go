package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := LoadConfig()

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var handler http.Handler
	switch cmd {
	case "serve":
		prepareSitesDir(cfg.SitesDir)
		handler = rootHandler(cfg, appHandler(cfg), sitesHandler(cfg))
	case "serve-app":
		prepareSitesDir(cfg.SitesDir)
		handler = appHandler(cfg)
	case "serve-sites":
		handler = sitesHandler(cfg)
	default:
		log.Fatalf("unknown command %q (valid: serve, serve-app, serve-sites)", cmd)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	log.Printf("%s listening on %s", cmd, cfg.Addr)
	if err := run(srv); err != nil {
		log.Fatal(err)
	}
}

func run(srv *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return err // the server died by itself
	case <-ctx.Done():
		log.Println("stop signal received, shutting down")
	}

	// Ongoing requests have 15 seconds to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
