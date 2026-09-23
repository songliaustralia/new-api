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
	"fmt"
	"strconv"
	"strings"
)

// byokGroupPrefix namespaces every self-service BYOK routing group so it can be
// recognized and re-derived from a user id alone, without a lookup table.
const byokGroupPrefix = "byok-u"

// BuildByokGroup returns the private routing group name that isolates a single
// user's own upstream API keys (submitted through the self-service BYOK flow)
// from every other user's channels and from the shared/pooled channels.
func BuildByokGroup(userId int) string {
	return fmt.Sprintf("%s%d", byokGroupPrefix, userId)
}

// IsUserOwnedByokGroup reports whether groupName is userId's own private BYOK
// group. It is intentionally independent of the admin-configured
// UserUsableGroups/GroupSpecialUsableGroup lists (see IsUserSelectableGroup):
// those are shared, admin-curated lists meant to stay small and human-reviewed,
// while a BYOK group is generated per user and must never be exposed to anyone
// else. Callers that accept a user-supplied group name for a token (single
// group or auto-groups) MUST treat IsUserSelectableGroup(...) ||
// IsUserOwnedByokGroup(...) as the full authorization check.
func IsUserOwnedByokGroup(userId int, groupName string) bool {
	suffix, ok := strings.CutPrefix(groupName, byokGroupPrefix)
	if !ok {
		return false
	}
	id, err := strconv.Atoi(suffix)
	if err != nil {
		return false
	}
	return id == userId
}
