package auth

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	guest, err := a.createGuest(ctx, sql.NullInt64{})
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
