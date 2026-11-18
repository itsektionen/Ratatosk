package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/itsektionen/ratatosk/internal/config"
	"golang.org/x/oauth2"
)

const oidcCookie = "ratatosk_oidc"

type oidcClient struct {
	oauth       oauth2.Config
	verifier    *oidc.IDTokenVerifier
	adminGroups []string
}

// roleFor makes admins of the members of an admin group, everyone else a member
func (c *oidcClient) roleFor(groups []string) Role {
	for _, g := range groups {
		if slices.Contains(c.adminGroups, g) {
			return RoleAdmin
		}
	}
	return RoleMember
}

func newOIDC(ctx context.Context, cfg config.Config) (*oidcClient, error) {
	if cfg.OIDCIssuer == "" {
		return nil, nil
	}
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, err
	}
	return &oidcClient{
		oauth: oauth2.Config{
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.PublicURL + "/api/auth/callback",
			Scopes:       []string{oidc.ScopeOpenID, "profile"},
		},
		verifier:    provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		adminGroups: cfg.OIDCAdminGroups,
	}, nil
}

// login sends the browser to the identity provider. The state, nonce and PKCE
// verifier are kept in a short-lived cookie until the provider sends it back
func (a *Auth) login(w http.ResponseWriter, r *http.Request) {
	if a.oidc == nil {
		http.Error(w, "login is not configured", http.StatusNotFound)
		return
	}
	state, nonce, verifier := oauth2.GenerateVerifier(), oauth2.GenerateVerifier(), oauth2.GenerateVerifier()
	http.SetCookie(w, &http.Cookie{
		Name:     a.cookieName(oidcCookie),
		Value:    state + "." + nonce + "." + verifier,
		Path:     "/", // __Host- needs it
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		Secure:   a.secure(),
		SameSite: http.SameSiteLaxMode,
	})
	url := a.oidc.oauth.AuthCodeURL(state,
		oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, url, http.StatusFound)
}

func (a *Auth) callback(w http.ResponseWriter, r *http.Request) {
	if a.oidc == nil {
		http.Error(w, "login is not configured", http.StatusNotFound)
		return
	}
	c, err := r.Cookie(a.cookieName(oidcCookie))
	if err != nil {
		http.Error(w, "login expired, try again", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     a.cookieName(oidcCookie),
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.secure(),
	})

	parts := strings.Split(c.Value, ".")
	if len(parts) != 3 ||
		subtle.ConstantTimeCompare([]byte(parts[0]), []byte(r.URL.Query().Get("state"))) != 1 {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	nonce, verifier := parts[1], parts[2]

	tok, err := a.oidc.oauth.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		http.Error(w, "login failed", http.StatusBadGateway)
		return
	}
	rawID, _ := tok.Extra("id_token").(string)
	idToken, err := a.oidc.verifier.Verify(r.Context(), rawID)
	if err != nil || subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonce)) != 1 {
		http.Error(w, "login failed", http.StatusBadGateway)
		return
	}
	var claims struct {
		Name     string   `json:"name"`
		Username string   `json:"preferred_username"`
		Groups   []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "login failed", http.StatusBadGateway)
		return
	}
	name := claims.Name
	if name == "" {
		name = claims.Username
	}
	if name == "" {
		name = idToken.Subject
	}

	u, err := a.upsertOIDCUser(r.Context(), idToken.Subject, name, a.oidc.roleFor(claims.Groups))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	a.startSession(w, r, u)
}
