package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestBootDataIsEmbeddedInHTML(t *testing.T) {
	h := Handler(func(*http.Request) ([]byte, error) { return []byte(`{"a":1}`), nil })
	for path, status := range map[string]int{"/": http.StatusOK, "/nope": http.StatusNotFound} {
		rec := get(h, path)
		if rec.Code != status {
			t.Errorf("%s: status %d, want %d", path, rec.Code, status)
		}
		head, _, _ := strings.Cut(rec.Body.String(), "</head>")
		if !strings.HasSuffix(head, `<script type="application/json" id="boot">{"a":1}</script>`) {
			t.Errorf("%s: boot data is not at the end of <head>", path)
		}
		if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
			t.Errorf("%s: per-user page is cacheable: %q", path, cc)
		}
	}
}

func TestPageStillServedWhenBootFails(t *testing.T) {
	h := Handler(func(*http.Request) ([]byte, error) { return nil, errors.New("db down") })
	rec := get(h, "/")
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), `id="boot"`) {
		t.Errorf("status %d, boot tag present=%v", rec.Code, strings.Contains(rec.Body.String(), `id="boot"`))
	}
}
