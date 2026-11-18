package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/itsektionen/ratatosk/internal/auth/authdb"
)

var (
	errInvalid  = errors.New("invalid or expired token")
	errNickname = errors.New("nickname must be 1 to 32 characters")
)

// cleanNickname trims name and checks that it is fit to show to others
func cleanNickname(name string) (string, error) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	if n < 1 || n > 32 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", errNickname
	}
	return name, nil
}

func newToken() (token string, hash []byte) {
	b := make([]byte, 32)
	rand.Read(b) // can panic, never returns error
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token)
}

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

func toUser(id int64, name, role string) User {
	return User{ID: id, Name: name, Role: Role(role)}
}

func (a *Auth) upsertOIDCUser(ctx context.Context, sub, name string, role Role) (User, error) {
	r, err := a.q.UpsertOIDCUser(ctx, authdb.UpsertOIDCUserParams{
		Name:      name,
		Role:      string(role),
		OidcSub:   sql.NullString{String: sub, Valid: true},
		CreatedAt: time.Now().Unix(),
	})
	return toUser(r.ID, r.Name, r.Role), err
}

func (a *Auth) createGuest(ctx context.Context, invitedBy sql.NullInt64, name string) (User, error) {
	r, err := a.q.CreateGuest(ctx, authdb.CreateGuestParams{
		Name:      name,
		Role:      string(RoleGuest),
		InvitedBy: invitedBy,
		CreatedAt: time.Now().Unix(),
	})
	return toUser(r.ID, r.Name, r.Role), err
}

func (a *Auth) createSession(ctx context.Context, userID int64) (string, error) {
	token, hash := newToken()
	now := time.Now()
	// Clean up expired rows as we go
	a.q.DeleteExpiredSessions(ctx, now.Unix())
	a.q.DeleteExpiredInvites(ctx, now.Unix())
	err := a.q.CreateSession(ctx, authdb.CreateSessionParams{
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: now.Add(sessionTTL).Unix(),
	})
	return token, err
}

func (a *Auth) sessionUser(ctx context.Context, token string) (User, error) {
	r, err := a.q.GetSessionUser(ctx, authdb.GetSessionUserParams{
		TokenHash: hashToken(token),
		Now:       time.Now().Unix(),
	})
	return toUser(r.ID, r.Name, r.Role), err
}

func (a *Auth) deleteSession(ctx context.Context, token string) error {
	return a.q.DeleteSession(ctx, hashToken(token))
}

// createInvite makes a link token. If forUser is set it is a share link that
// signs in as them, otherwise it is a guest invite
func (a *Auth) createInvite(ctx context.Context, createdBy, forUser int64, ttl time.Duration) (string, time.Time, error) {
	token, hash := newToken()
	expires := time.Now().Add(ttl)
	err := a.q.CreateInvite(ctx, authdb.CreateInviteParams{
		TokenHash: hash,
		UserID:    sql.NullInt64{Int64: forUser, Valid: forUser != 0},
		CreatedBy: sql.NullInt64{Int64: createdBy, Valid: true},
		ExpiresAt: expires.Unix(),
	})
	return token, expires, err
}

// redeemInvite returns the user the link signs in. For a guest invite that is a new guest
// called name, and errNickname if name is unfit. Share links ignore name
func (a *Auth) redeemInvite(ctx context.Context, token, name string) (User, error) {
	inv, err := a.q.GetInvite(ctx, authdb.GetInviteParams{
		TokenHash: hashToken(token),
		Now:       time.Now().Unix(),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, errInvalid
	}
	if err != nil {
		return User{}, err
	}
	if !inv.UserID.Valid {
		name, err := cleanNickname(name)
		if err != nil {
			return User{}, err
		}
		return a.createGuest(ctx, inv.CreatedBy, name)
	}
	r, err := a.q.GetUser(ctx, inv.UserID.Int64)
	return toUser(r.ID, r.Name, r.Role), err
}
