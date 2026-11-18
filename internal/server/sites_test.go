package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/itsektionen/ratatosk/internal/config"
)

// Sandboxed previews have a null origin, so they need CORS to load a site's own files
func TestSitesAllowCrossOriginReads(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "demo", "index.html"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := Sites(config.Config{SitesDir: dir, BaseDomain: "tvs.example"})

	for path, status := range map[string]int{"/": 200, "/missing.png": 404} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "demo.tvs.example"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != status || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("%s: status %d, CORS=%q", path, rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
		}
	}
}
