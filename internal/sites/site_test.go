package sites

import "testing"

func TestSiteOwnership(t *testing.T) {
	for _, tc := range []struct {
		name   string
		site   Site
		userID int64
		owned  bool
		by     bool
	}{
		{"owned by the user", Site{OwnerID: 2}, 2, true, true},
		{"owned by someone else", Site{OwnerID: 2}, 3, true, false},
		{"unowned, any user", Site{}, 2, false, false},
		// A zero-value user must not match a site with no owner
		{"unowned, zero user", Site{}, 0, false, false},
	} {
		if got := tc.site.Owned(); got != tc.owned {
			t.Errorf("%s: Owned=%v, want %v", tc.name, got, tc.owned)
		}
		if got := tc.site.OwnedBy(tc.userID); got != tc.by {
			t.Errorf("%s: OwnedBy(%d)=%v, want %v", tc.name, tc.userID, got, tc.by)
		}
	}
}
