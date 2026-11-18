package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInviteRequiresLifetime(t *testing.T) {
	a, _ := newTestAuth(t)
	a.cfg.PublicURL = "http://test"
	admin, err := a.upsertOIDCUser(context.Background(), "sub", "Admin", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body string
		want int
	}{
		{`{"hours":1}`, http.StatusOK},
		{`{"hours":5}`, http.StatusOK},
		{"", http.StatusBadRequest},
		{`{}`, http.StatusBadRequest},
		{`{"hours":0}`, http.StatusBadRequest},
		{`{"hours":-1}`, http.StatusBadRequest},
		{`{"hours":9223372036854775807}`, http.StatusBadRequest},
		{fmt.Sprintf(`{"hours":%d}`, maxInviteHours), http.StatusOK},
	} {
		req := httptest.NewRequest(http.MethodPost, "/invites", strings.NewReader(tc.body))
		req = req.WithContext(withUser(req.Context(), admin))
		rec := httptest.NewRecorder()
		a.invite(rec, req)
		if rec.Code != tc.want {
			t.Errorf("body %q: got %d, want %d", tc.body, rec.Code, tc.want)
		}
	}
}

// SSO users must not be able to share their session
func TestShareIsGuestsOnly(t *testing.T) {
	ctx := context.Background()
	a, _ := newTestAuth(t)
	a.cfg.PublicURL = "http://test"
	guest, err := a.createGuest(ctx, sql.NullInt64{}, "Guest")
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []struct {
		user User
		want int
	}{
		{guest, http.StatusOK},
		{User{ID: guest.ID, Role: RoleMember}, http.StatusForbidden},
		{User{ID: guest.ID, Role: RoleAdmin}, http.StatusForbidden},
	} {
		req := httptest.NewRequest(http.MethodPost, "/share", nil)
		req = req.WithContext(withUser(req.Context(), u.user))
		rec := httptest.NewRecorder()
		a.share(rec, req)
		if rec.Code != u.want {
			t.Errorf("role %q: got %d, want %d", u.user.Role, rec.Code, u.want)
		}
	}
}

func userCount(t *testing.T, db *sql.DB) (n int) {
	t.Helper()
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func hasSession(a *Auth, rec *httptest.ResponseRecorder) bool {
	for _, c := range rec.Result().Cookies() {
		if c.Name == a.cookieName(sessionCookie) && c.Value != "" {
			return true
		}
	}
	return false
}

// A guest has to pick a nickname before they get a session, or even exist
func TestGuestInviteAsksForNickname(t *testing.T) {
	ctx := context.Background()
	a, database := newTestAuth(t)
	inviter, err := a.upsertOIDCUser(ctx, "sub", "Alice", RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := a.createInvite(ctx, inviter.ID, 0, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := a.Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/join?token="+token, nil))
	if loc := rec.Header().Get("Location"); rec.Code != http.StatusSeeOther || loc != "/?invite="+token {
		t.Errorf("GET /join: status %d, location %q, want a redirect to the nickname prompt", rec.Code, loc)
	}
	if hasSession(a, rec) || userCount(t, database) != 1 {
		t.Errorf("opening the link must not sign anyone in or create a guest")
	}

	for _, name := range []string{"", "   ", "a\nb", strings.Repeat("x", 33)} {
		rec := httptest.NewRecorder()
		body, _ := json.Marshal(map[string]string{"token": token, "name": name})
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/join", bytes.NewReader(body)))
		if rec.Code != http.StatusBadRequest || hasSession(a, rec) || userCount(t, database) != 1 {
			t.Errorf("name %q: status %d, want 400 with no session or guest", name, rec.Code)
		}
	}

	rec = httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"token": token, "name": "  Nick  "})
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/join", bytes.NewReader(body)))
	if rec.Code != http.StatusNoContent || !hasSession(a, rec) {
		t.Fatalf("POST /join: status %d, session=%v, want 204 with a session", rec.Code, hasSession(a, rec))
	}
	var name, role string
	if err := database.QueryRow(`SELECT name, role FROM users WHERE id != ?`, inviter.ID).Scan(&name, &role); err != nil {
		t.Fatal(err)
	}
	if name != "Nick" || role != string(RoleGuest) {
		t.Errorf("guest is %q (%s), want Nick (guest)", name, role)
	}

	rec = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]string{"token": "nope", "name": "Nick"})
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/join", bytes.NewReader(body)))
	if rec.Code != http.StatusGone {
		t.Errorf("bad token: status %d, want %d", rec.Code, http.StatusGone)
	}
}

// Sharing a session to another device has no nickname step, the user already has a name
func TestShareLinkSignsInWithoutNickname(t *testing.T) {
	ctx := context.Background()
	a, _ := newTestAuth(t)
	u, err := a.createGuest(ctx, sql.NullInt64{}, "Nick")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := a.createInvite(ctx, u.ID, u.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/join?token="+token, nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" || !hasSession(a, rec) {
		t.Errorf("status %d, location %q, session=%v, want a redirect to / with a session",
			rec.Code, rec.Header().Get("Location"), hasSession(a, rec))
	}
}

// On https a site on a subdomain must not be able to replace our cookies, which the
// __Host- prefix guarantees as long as they are Secure, have Path=/ and no Domain
func TestCookiesAreHostPrefixedOnHTTPS(t *testing.T) {
	ctx := context.Background()
	a, _ := newTestAuth(t)
	a.cfg.PublicURL = "https://example.com"
	u, err := a.createGuest(ctx, sql.NullInt64{}, "Nick")
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := a.createInvite(ctx, u.ID, u.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/join?token="+token, nil))

	var found bool
	for _, c := range rec.Result().Cookies() {
		if c.Name != "__Host-"+sessionCookie {
			continue
		}
		found = true
		if !c.Secure || c.Path != "/" || c.Domain != "" || !c.HttpOnly {
			t.Errorf("cookie %+v must be Secure, HttpOnly, Path=/ and have no Domain", c)
		}
	}
	if !found {
		t.Errorf("no __Host- session cookie among %v", rec.Result().Cookies())
	}
}

// A cookie with the unprefixed name, which a site could have set for the whole domain, is ignored
func TestUnprefixedSessionCookieIgnoredOnHTTPS(t *testing.T) {
	ctx := context.Background()
	a, _ := newTestAuth(t)
	a.cfg.PublicURL = "https://example.com"
	u, err := a.createGuest(ctx, sql.NullInt64{}, "Nick")
	if err != nil {
		t.Fatal(err)
	}
	session, _, err := a.createSession(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{sessionCookie: false, "__Host-" + sessionCookie: true} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: name, Value: session})
		var got bool
		a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, got = FromContext(r.Context())
		})).ServeHTTP(httptest.NewRecorder(), req)
		if got != want {
			t.Errorf("cookie %q: signed in=%v, want %v", name, got, want)
		}
	}
}
