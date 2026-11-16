package auth

import "context"

type Role string

const (
	// Can do anything
	RoleAdmin Role = "admin"
	// Can edit their own sites, and can invite guests
	RoleMember Role = "member"
	// Can edit their own sites
	RoleGuest Role = "guest"
)

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role Role   `json:"role"`
}

// CanEdit reports whether u may modify a site. ownedByU is whether u owns it
func (u User) CanEdit(ownedByU bool) bool {
	return u.Role == RoleAdmin || ownedByU
}

// CanInvite reports whether u may create guest invite links
func (u User) CanInvite() bool {
	return u.Role == RoleAdmin || u.Role == RoleMember
}

type userKey struct{}

func withUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

// FromContext returns the signed-in user, if any
func FromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userKey{}).(User)
	return u, ok
}
