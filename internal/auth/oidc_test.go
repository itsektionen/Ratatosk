package auth

import "testing"

func TestRoleFor(t *testing.T) {
	c := &oidcClient{adminGroups: []string{"staff", "ops"}}
	for _, tc := range []struct {
		groups []string
		want   Role
	}{
		{nil, RoleMember},
		{[]string{"students"}, RoleMember},
		{[]string{"students", "ops"}, RoleAdmin},
		{[]string{"staff"}, RoleAdmin},
		{[]string{"Staff"}, RoleMember}, // group names are case-sensitive
	} {
		if got := c.roleFor(tc.groups); got != tc.want {
			t.Errorf("roleFor(%v)=%q, want %q", tc.groups, got, tc.want)
		}
	}
	// With no admin groups configured nobody is an admin
	if got := (&oidcClient{}).roleFor([]string{"staff"}); got != RoleMember {
		t.Errorf("no admin groups: got %q, want %q", got, RoleMember)
	}
}
