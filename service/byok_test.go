/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildByokGroup(t *testing.T) {
	assert.Equal(t, "byok-u42", BuildByokGroup(42))
	assert.Equal(t, "byok-u0", BuildByokGroup(0))
}

// TestIsUserOwnedByokGroup guards the one property the whole BYOK design
// leans on: a user can only ever be authorized for their own derived group
// name, never another user's (including by malformed/spoofed suffixes).
func TestIsUserOwnedByokGroup(t *testing.T) {
	tests := []struct {
		name      string
		userId    int
		groupName string
		want      bool
	}{
		{"own group matches", 42, "byok-u42", true},
		{"another user's group is rejected", 42, "byok-u43", false},
		{"a differently-numbered user's group is rejected", 1, "byok-u10", false},
		{"unrelated group name is rejected", 42, "default", false},
		{"admin-curated group name is rejected", 42, "vip", false},
		{"empty group name is rejected", 42, "", false},
		{"bare prefix with no id is rejected", 42, "byok-u", false},
		{"non-numeric suffix is rejected", 42, "byok-uabc", false},
		{"negative-looking suffix does not falsely match", 42, "byok-u-42", false},
		{"decimal suffix is rejected", 42, "byok-u42.0", false},
		{"trailing garbage after the id is rejected", 42, "byok-u42x", false},
		{"leading zero still parses to the same id", 42, "byok-u042", true},
		{"case-sensitive prefix: differently-cased prefix is rejected", 42, "BYOK-u42", false},
		{"user id zero owns its own group", 0, "byok-u0", true},
		{"user id zero does not own another id's group", 0, "byok-u1", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsUserOwnedByokGroup(tc.userId, tc.groupName))
		})
	}
}

// TestIsUserOwnedByokGroup_RoundTrip pins BuildByokGroup and
// IsUserOwnedByokGroup together: whatever the builder produces for a user
// must always be recognized as that same user's own group.
func TestIsUserOwnedByokGroup_RoundTrip(t *testing.T) {
	for _, userId := range []int{0, 1, 42, 999999} {
		group := BuildByokGroup(userId)
		assert.True(t, IsUserOwnedByokGroup(userId, group))
		assert.False(t, IsUserOwnedByokGroup(userId+1, group))
	}
}
