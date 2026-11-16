package auth

import (
	"net/http"
	"strings"
	"time"
)

const sessionCookie = "ratatosk_session"

// The cookie has no Domain attribute, so browsers only send it to the app's own
// host and never to <site>.BASE_DOMAIN, where uploaded (untrusted) JS runs
func (a *Auth) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   strings.HasPrefix(a.cfg.PublicURL, "https://"),
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) clearSessionCookie(w http.ResponseWriter) {
	a.setSessionCookie(w, "", time.Unix(0, 0))
}

// Middleware loads the signed-in user, if any, into the request context
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookie); err == nil {
			if u, err := a.sessionUser(r.Context(), c.Value); err == nil {
				r = r.WithContext(withUser(r.Context(), u))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); !ok {
			http.Error(w, "sign in required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireInviter lets through the users who can create guest invites
func RequireInviter(next http.Handler) http.Handler {
	return RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, _ := FromContext(r.Context()); !u.CanInvite() {
			http.Error(w, "members and admins only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}
