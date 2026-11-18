package app

import (
	"net/http"

	"github.com/itsektionen/ratatosk/internal/config"
)

// sameOriginOnly rejects requests that change things unless the app's own pages made them.
// Sites run on subdomains, which browsers count as the same site, so SameSite cookies
// don't stop a site from making a visitor's browser post to the API
func sameOriginOnly(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			if site := r.Header.Get("Sec-Fetch-Site"); site != "" {
				if site != "same-origin" && site != "none" {
					http.Error(w, "cross-site request", http.StatusForbidden)
					return
				}
			} else if origin := r.Header.Get("Origin"); origin != "" && origin != cfg.PublicURL {
				// Browsers without Sec-Fetch-Site. Tools like curl send neither header
				http.Error(w, "cross-site request", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// denyFraming stops other pages, including uploaded sites, from embedding the app
func denyFraming(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
		w.Header().Set("X-Frame-Options", "DENY") // for old browsers
		next.ServeHTTP(w, r)
	})
}
