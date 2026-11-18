package auth

import (
	"cmp"
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/httpx"
	"golang.org/x/oauth2"
)

const oidcCookie = "ratatosk_oidc"

type oidcClient struct {
	oauth       oauth2.Config
	verifier    *oidc.IDTokenVerifier
	adminGroups []string
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

func (c *oidcClient) roleFor(groups []string) Role {
	for _, g := range groups {
		if slices.Contains(c.adminGroups, g) {
			return RoleAdmin
		}
	}
	return RoleMember
}

type identity struct {
	sub  string
	name string
	role Role
}

// identify trades the code the provider sent back for who signed in
func (c *oidcClient) identify(ctx context.Context, code, nonce, verifier string) (identity, error) {
	tok, err := c.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return identity{}, err
	}
	rawID, _ := tok.Extra("id_token").(string)
	idToken, err := c.verifier.Verify(ctx, rawID)
	if err != nil {
		return identity{}, err
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonce)) != 1 {
		return identity{}, errors.New("nonce mismatch")
	}
	var claims struct {
		Name     string   `json:"name"`
		Username string   `json:"preferred_username"`
		Groups   []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return identity{}, err
	}
	return identity{
		sub:  idToken.Subject,
		name: cmp.Or(claims.Name, claims.Username, idToken.Subject),
		role: c.roleFor(claims.Groups),
	}, nil
}

// login sends the browser to the identity provider. The state, nonce and PKCE
// verifier are kept in a short-lived cookie until the provider sends it back
func (a *Auth) login(w http.ResponseWriter, r *http.Request) {
	state, nonce, verifier := oauth2.GenerateVerifier(), oauth2.GenerateVerifier(), oauth2.GenerateVerifier()
	a.setCookie(w, oidcCookie, state+"."+nonce+"."+verifier, 10*time.Minute)
	url := a.oidc.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, url, http.StatusFound)
}

func (a *Auth) callback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(a.cookieName(oidcCookie))
	if err != nil {
		http.Error(w, "login expired, try again", http.StatusBadRequest)
		return
	}
	a.deleteCookie(w, oidcCookie)

	query := r.URL.Query()
	parts := strings.Split(c.Value, ".")
	if len(parts) != 3 || subtle.ConstantTimeCompare([]byte(parts[0]), []byte(query.Get("state"))) != 1 {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	id, err := a.oidc.identify(r.Context(), query.Get("code"), parts[1], parts[2])
	if err != nil {
		slog.Warn("oidc login failed", "err", err)
		http.Error(w, "login failed", http.StatusBadGateway)
		return
	}

	u, err := a.upsertOIDCUser(r.Context(), id.sub, id.name, id.role)
	if err != nil {
		httpx.InternalError(w, "upsert user", err)
		return
	}
	a.signInAndRedirect(w, r, u)
}
