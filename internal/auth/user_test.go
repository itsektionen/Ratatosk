package auth

import "testing"

func TestPermissions(t *testing.T) {
	admin := User{ID: 1, Role: RoleAdmin}
	member := User{ID: 2, Role: RoleMember}
	guest := User{ID: 3, Role: RoleGuest}

	for _, tc := range []struct {
		name      string
		u         User
		owned     bool // whether u owns the site
		canEdit   bool
		canInvite bool
	}{
		{"admin edits a site they don't own", admin, false, true, true},
		{"admin edits own site", admin, true, true, true},
		{"member edits a site they don't own", member, false, false, true},
		{"member edits own site", member, true, true, true},
		{"guest edits own site", guest, true, true, false},
		{"guest edits a site they don't own", guest, false, false, false},
		{"user without a role can't edit others'", User{}, false, false, false},
	} {
		if got := tc.u.CanEdit(tc.owned); got != tc.canEdit {
			t.Errorf("%s: CanEdit=%v, want %v", tc.name, got, tc.canEdit)
		}
		if got := tc.u.CanInvite(); got != tc.canInvite {
			t.Errorf("%s: CanInvite=%v, want %v", tc.name, got, tc.canInvite)
		}
	}
}
