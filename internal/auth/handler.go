package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
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

// startSession signs the browser in as u and sends it to the UI
func (a *Auth) startSession(w http.ResponseWriter, r *http.Request, u User) {
	token, expires, err := a.createSession(r.Context(), u.ID)
	if err != nil {
		slog.Error("create session", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	a.setSessionCookie(w, token, expires)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// me reports who is signed in. user is null when nobody is
func (a *Auth) me(w http.ResponseWriter, r *http.Request) {
	resp := struct {
		User         *User `json:"user"`
		LoginEnabled bool  `json:"loginEnabled"`
		DevLogin     bool  `json:"devLogin"`
	}{LoginEnabled: a.oidc != nil, DevLogin: a.cfg.DevLogin}
	if u, ok := FromContext(r.Context()); ok {
		resp.User = &u
	}
	writeJSON(w, resp)
}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
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

// join redeems an invite or share link, which is what the QR code points at
func (a *Auth) join(w http.ResponseWriter, r *http.Request) {
	u, err := a.redeemInvite(r.Context(), r.URL.Query().Get("token"))
	if errors.Is(err, errInvalid) {
		http.Error(w, "this link is invalid or has expired", http.StatusGone)
		return
	}
	if err != nil {
		slog.Error("redeem invite", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	a.startSession(w, r, u)
}

func (a *Auth) devLogin(w http.ResponseWriter, r *http.Request) {
	u, err := a.upsertOIDCUser(r.Context(), "dev", "Dev Admin", RoleAdmin)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	a.startSession(w, r, u)
}
