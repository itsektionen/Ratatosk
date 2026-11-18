package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/itsektionen/ratatosk/internal/config"
	"github.com/itsektionen/ratatosk/internal/db"
)

func newTestAuth(t *testing.T) (*Auth, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	database, err := db.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	a, err := New(ctx, config.Config{}, database)
	if err != nil {
		t.Fatal(err)
	}
	return a, database
}

// A user removed from the admin group must lose the role at their next login
func TestUpsertOIDCUserUpdatesRole(t *testing.T) {
	ctx := context.Background()
	a, _ := newTestAuth(t)

	first, err := a.upsertOIDCUser(ctx, "sub-1", "Alice", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	again, err := a.upsertOIDCUser(ctx, "sub-1", "Alice", RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID {
		t.Errorf("a second login created user %d, want the same user %d", again.ID, first.ID)
	}
	if again.Role != RoleMember {
		t.Errorf("role after demotion=%q, want %q", again.Role, RoleMember)
	}
}

func TestRedeemInviteRecordsInviter(t *testing.T) {
	ctx := context.Background()
	a, database := newTestAuth(t)

	inviter, err := a.upsertOIDCUser(ctx, "sub-1", "Alice", RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := a.createInvite(ctx, inviter.ID, 0, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	guest, err := a.redeemInvite(ctx, token, "Nick")
	if err != nil {
		t.Fatal(err)
	}
	if guest.Name != "Nick" {
		t.Errorf("name=%q, want %q", guest.Name, "Nick")
	}
	if guest.Role != RoleGuest {
		t.Fatalf("role=%q, want %q", guest.Role, RoleGuest)
	}

	var invitedBy sql.NullInt64
	if err := database.QueryRow(`SELECT invited_by FROM users WHERE id = ?`, guest.ID).Scan(&invitedBy); err != nil {
		t.Fatal(err)
	}
	if !invitedBy.Valid || invitedBy.Int64 != inviter.ID {
		t.Errorf("invited_by=%v, want %d", invitedBy, inviter.ID)
	}

	// OIDC users weren't invited by anyone
	if err := database.QueryRow(`SELECT invited_by FROM users WHERE id = ?`, inviter.ID).Scan(&invitedBy); err != nil {
		t.Fatal(err)
	}
	if invitedBy.Valid {
		t.Errorf("an OIDC user has invited_by=%v, want NULL", invitedBy)
	}
}

// A share link signs in an existing user, it must not create a guest
func TestRedeemShareLinkCreatesNoUser(t *testing.T) {
	ctx := context.Background()
	a, database := newTestAuth(t)

	u, err := a.upsertOIDCUser(ctx, "sub-1", "Alice", RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := a.createInvite(ctx, u.ID, u.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.redeemInvite(ctx, token, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Errorf("signed in as user %d, want %d", got.ID, u.ID)
	}
	if n := userCount(t, database); n != 1 {
		t.Errorf("users=%d, want 1", n)
	}
}
