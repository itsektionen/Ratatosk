package auth

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/itsektionen/ratatosk/internal/httpx"
)

const (
	// How long the session cookie keeps someone signed in for
	sessionTTL = 30 * 24 * time.Hour
	// How long session-sharing links last
	shareLinkTTL = 10 * time.Minute
	// Overflow prevention for user-pickable duration
	maxInviteHours = math.MaxInt64 / int64(time.Hour)
)

// Handler serves the /api/auth routes. It expects Middleware to have run
func (a *Auth) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/me", a.me)
	r.Get("/join", a.join)
	r.Post("/join", a.joinAsGuest)
	r.Post("/logout", a.logout)
	r.With(RequireInviter).Post("/invites", a.invite)
	r.With(RequireUser).Post("/share", a.share)
	if a.oidc != nil {
		r.Get("/login", a.login)
		r.Get("/callback", a.callback)
	}
	if a.cfg.DevLogin {
		slog.Warn("DEV_LOGIN is enabled, anyone can sign in as an admin")
		r.Post("/dev-login", a.devLogin)
	}
	return r
}

// signIn gives the browser a session as u. It writes the error response and
// returns false if that fails
func (a *Auth) signIn(w http.ResponseWriter, r *http.Request, u User) bool {
	token, err := a.createSession(r.Context(), u.ID)
	if err != nil {
		httpx.InternalError(w, "create session", err)
		return false
	}
	a.setCookie(w, sessionCookie, token, sessionTTL)
	return true
}

func (a *Auth) signInAndRedirect(w http.ResponseWriter, r *http.Request, u User) {
	if a.signIn(w, r, u) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Me is who is signed in, and how others can sign in
type Me struct {
	User         *User `json:"user"` // null when nobody is
	LoginEnabled bool  `json:"loginEnabled"`
	DevLogin     bool  `json:"devLogin"`
}

func (a *Auth) Me(ctx context.Context) Me {
	me := Me{LoginEnabled: a.oidc != nil, DevLogin: a.cfg.DevLogin}
	if u, ok := FromContext(ctx); ok {
		me.User = &u
	}
	return me
}

func (a *Auth) me(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, a.Me(r.Context()))
}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.cookieName(sessionCookie)); err == nil {
		a.deleteSession(r.Context(), c.Value)
	}
	a.deleteCookie(w, sessionCookie)
	w.WriteHeader(http.StatusNoContent)
}

// invite makes a link that lets anyone become a guest
func (a *Auth) invite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Hours int64 `json:"hours"`
	}
	if !httpx.ReadJSON(w, r, &body) {
		return
	}
	if body.Hours <= 0 || body.Hours > maxInviteHours {
		http.Error(w, "hours out of range", http.StatusBadRequest)
		return
	}
	u, _ := FromContext(r.Context())
	a.writeLink(w, r, u.ID, 0, time.Duration(body.Hours)*time.Hour)
}

// share makes a short-lived link that signs another device in as the caller.
// Guests only, everyone else signs in through SSO
func (a *Auth) share(w http.ResponseWriter, r *http.Request) {
	u, _ := FromContext(r.Context())
	if u.Role != RoleGuest {
		http.Error(w, "sign in through SSO on the other device instead", http.StatusForbidden)
		return
	}
	a.writeLink(w, r, u.ID, u.ID, shareLinkTTL)
}

// writeLink creates an invite (see createInvite) and responds with its URL
func (a *Auth) writeLink(w http.ResponseWriter, r *http.Request, createdBy, forUser int64, ttl time.Duration) {
	token, expires, err := a.createInvite(r.Context(), createdBy, forUser, ttl)
	if err != nil {
		httpx.InternalError(w, "create invite", err)
		return
	}
	httpx.WriteJSON(w, struct {
		URL     string    `json:"url"`
		Expires time.Time `json:"expires"`
	}{a.cfg.PublicURL + "/api/auth/join?token=" + token, expires})
}

// join is what invite and share links point at. A share link signs in right away.
// A guest invite needs a nickname first, so it sends the browser to the UI to ask
// for one, which then posts it to joinAsGuest along with the token
func (a *Auth) join(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	u, err := a.redeemInvite(r.Context(), token, "")
	if errors.Is(err, errNickname) {
		http.Redirect(w, r, "/?invite="+url.QueryEscape(token), http.StatusSeeOther)
		return
	}
	if err != nil {
		joinError(w, err)
		return
	}
	a.signInAndRedirect(w, r, u)
}

func (a *Auth) joinAsGuest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if !httpx.ReadJSON(w, r, &body) {
		return
	}
	u, err := a.redeemInvite(r.Context(), body.Token, body.Name)
	if err != nil {
		joinError(w, err)
		return
	}
	if a.signIn(w, r, u) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func joinError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalid):
		http.Error(w, "this link is invalid or has expired", http.StatusGone)
	case errors.Is(err, errNickname):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		httpx.InternalError(w, "redeem invite", err)
	}
}

func (a *Auth) devLogin(w http.ResponseWriter, r *http.Request) {
	u, err := a.upsertOIDCUser(r.Context(), "dev", "Dev Admin", RoleAdmin)
	if err != nil {
		httpx.InternalError(w, "upsert user", err)
		return
	}
	a.signInAndRedirect(w, r, u)
}
