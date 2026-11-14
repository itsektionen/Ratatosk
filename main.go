package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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

func rootHandler(cfg Config, app, sites http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hostname(r) == cfg.BaseDomain {
			app.ServeHTTP(w, r)
			return
		}
		sites.ServeHTTP(w, r)
	})
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

const (
	// 0o755: owner can read/write/enter, everyone else can read/enter.
	dirPerm os.FileMode = 0o755
	// 0o644: owner can read/write, everyone else can read.
	filePerm os.FileMode = 0o644
)

func prepareSitesDir(dir string) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		log.Fatal(err)
	}
	// <SitesDir>/.upload-*/ hold sites currently being extracted
	// They might not get deleted normally if the process is killed or crashes
	for _, pat := range []string{".upload-*", ".old-*"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pat))
		for _, m := range matches {
			os.RemoveAll(m)
		}
	}
}
