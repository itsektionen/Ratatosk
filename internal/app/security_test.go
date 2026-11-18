package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/itsektionen/ratatosk/internal/config"
)

func TestSameOriginOnly(t *testing.T) {
	cfg := config.Config{PublicURL: "https://tvs.example"}
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h := sameOriginOnly(cfg)(ok)

	for _, tc := range []struct {
		name    string
		method  string
		headers map[string]string
		want    int
	}{
		{"reading from a site is fine", "GET", map[string]string{"Sec-Fetch-Site": "same-site"}, 200},
		{"own page", "POST", map[string]string{"Sec-Fetch-Site": "same-origin"}, 200},
		{"a site on a subdomain", "POST", map[string]string{"Sec-Fetch-Site": "same-site"}, 403},
		{"another site", "DELETE", map[string]string{"Sec-Fetch-Site": "cross-site"}, 403},
		{"old browser, own page", "POST", map[string]string{"Origin": "https://tvs.example"}, 200},
		{"old browser, a site", "POST", map[string]string{"Origin": "https://evil.tvs.example"}, 403},
		{"sandboxed page", "POST", map[string]string{"Origin": "null"}, 403},
		{"curl and such", "POST", nil, 200},
	} {
		req := httptest.NewRequest(tc.method, "/api/upload", nil)
		for k, v := range tc.headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, rec.Code, tc.want)
		}
	}
}

func TestDenyFraming(t *testing.T) {
	rec := httptest.NewRecorder()
	denyFraming(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if got := rec.Header().Get("Content-Security-Policy"); got != "frame-ancestors 'none'" {
		t.Errorf("CSP=%q", got)
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options is not DENY")
	}
}
