package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
)

// Short, since it signs in as the caller
const shareLinkTTL = 10 * time.Minute

// Overflow limit of time.Duration
const maxInviteHours = math.MaxInt64 / int64(time.Hour)

// Handler serves the /api/auth routes. It expects Middleware to have run
func (a *Auth) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/me", a.me)
	r.Get("/login", a.login)
	r.Get("/callback", a.callback)
	r.Get("/join", a.join)
	r.Post("/join", a.joinAsGuest)
	r.Post("/logout", a.logout)
	r.With(RequireInviter).Post("/invites", a.invite)
	r.With(RequireUser).Post("/share", a.share)
	if a.cfg.DevLogin {
		slog.Warn("DEV_LOGIN is enabled, anyone can sign in as an admin")
		r.Post("/dev-login", a.devLogin)
	}
	return r
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// signIn gives the browser a session as u. It writes the error response and
// returns false if that fails
func (a *Auth) signIn(w http.ResponseWriter, r *http.Request, u User) bool {
	token, expires, err := a.createSession(r.Context(), u.ID)
	if err != nil {
		slog.Error("create session", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return false
	}
	a.setSessionCookie(w, token, expires)
	return true
}

// startSession signs the browser in as u and sends it to the UI
func (a *Auth) startSession(w http.ResponseWriter, r *http.Request, u User) {
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
	writeJSON(w, a.Me(r.Context()))
}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.cookieName(sessionCookie)); err == nil {
		a.deleteSession(r.Context(), c.Value)
	}
	a.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// invite makes a link that lets anyone become a guest
func (a *Auth) invite(w http.ResponseWriter, r *http.Request) {
	u, _ := FromContext(r.Context())
	var body struct {
		Hours int64 `json:"hours"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Hours <= 0 || body.Hours > maxInviteHours {
		http.Error(w, "hours out of range", http.StatusBadRequest)
		return
	}
	a.respondWithLink(w, r, u.ID, 0, time.Duration(body.Hours)*time.Hour)
}

// share makes a short-lived link that signs another device in as the caller.
// Guests only, everyone else signs in through SSO
func (a *Auth) share(w http.ResponseWriter, r *http.Request) {
	u, _ := FromContext(r.Context())
	if u.Role != RoleGuest {
		http.Error(w, "sign in through SSO on the other device instead", http.StatusForbidden)
		return
	}
	a.respondWithLink(w, r, u.ID, u.ID, shareLinkTTL)
}

func (a *Auth) respondWithLink(w http.ResponseWriter, r *http.Request, createdBy, forUser int64, ttl time.Duration) {
	token, expires, err := a.createInvite(r.Context(), createdBy, forUser, ttl)
	if err != nil {
		slog.Error("create invite", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, struct {
		URL     string    `json:"url"`
		Expires time.Time `json:"expires"`
	}{a.cfg.PublicURL + "/api/auth/join?token=" + token, expires})
}

// join is what invite and share links point at. A share link signs in right away.
// A guest invite first sends the browser to the UI to ask for a nickname,
// which joinAsGuest then gets along with the token
func (a *Auth) join(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	needsName, err := a.inviteNeedsName(r.Context(), token)
	if needsName {
		http.Redirect(w, r, "/?invite="+url.QueryEscape(token), http.StatusSeeOther)
		return
	}
	if err == nil {
		var u User
		if u, err = a.redeemInvite(r.Context(), token, ""); err == nil {
			a.startSession(w, r, u)
			return
		}
	}
	a.joinError(w, err)
}

// joinAsGuest redeems a guest invite for a new guest with the given nickname
func (a *Auth) joinAsGuest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	u, err := a.redeemInvite(r.Context(), body.Token, body.Name)
	if err != nil {
		a.joinError(w, err)
		return
	}
	if a.signIn(w, r, u) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (a *Auth) joinError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalid):
		http.Error(w, "this link is invalid or has expired", http.StatusGone)
	case errors.Is(err, errNickname):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		slog.Error("redeem invite", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (a *Auth) devLogin(w http.ResponseWriter, r *http.Request) {
	u, err := a.upsertOIDCUser(r.Context(), "dev", "Dev Admin", RoleAdmin)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	a.startSession(w, r, u)
}
