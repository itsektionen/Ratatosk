package auth

import (
	"net/http"
	"strings"
	"time"
)

const sessionCookie = "ratatosk_session"

func (a *Auth) secure() bool {
	return strings.HasPrefix(a.cfg.PublicURL, "https://")
}

// cookieName adds the __Host- prefix on https. It makes browsers refuse a cookie
// that a site on a subdomain sets for the whole domain, so a site can't replace ours
func (a *Auth) cookieName(name string) string {
	if a.secure() {
		return "__Host-" + name
	}
	return name
}

// setCookie sets a cookie that scripts can't read and that lasts for ttl
func (a *Auth) setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.cookieName(name),
		Value:    value,
		Path:     "/", // __Host- needs it
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   a.secure(),
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) deleteCookie(w http.ResponseWriter, name string) {
	a.setCookie(w, name, "", -time.Second)
}

// Middleware loads the signed-in user, if any, into the request context
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(a.cookieName(sessionCookie)); err == nil {
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
