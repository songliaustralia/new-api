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
package controller

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupByokControllerTest gives each test its own in-memory database and a
// single user in the "default" group, mirroring the state SetByokKey expects
// to find (see controller/byok.go): a group-ratio table it can look up, and a
// user row GetUserGroups can resolve.
func setupByokControllerTest(t *testing.T, userID int) *model.User {
	t.Helper()

	db := openTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Channel{}))

	user := &model.User{
		Id:       userID,
		Username: fmt.Sprintf("byok-user-%d", userID),
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(user).Error)

	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})

	return user
}

func TestUserHasAnyByokChannel(t *testing.T) {
	user := setupByokControllerTest(t, 4201)
	group := service.BuildByokGroup(user.Id)

	has, err := userHasAnyByokChannel(user.Id)
	require.NoError(t, err)
	assert.False(t, has, "a user with no configured BYOK channel must report false")

	require.NoError(t, model.DB.Create(&model.Channel{
		Type:  constant.ChannelTypeAnthropic,
		Key:   "test-key",
		Group: group,
	}).Error)

	has, err = userHasAnyByokChannel(user.Id)
	require.NoError(t, err)
	assert.True(t, has, "a user with a configured BYOK channel must report true")
}

// TestGetUserGroupsExposesByokGroupOnlyOnceConfigured protects the fix for a
// bug found during BYOK deployment testing: the token-creation UI's group
// picker (web/src/features/keys/components/api-key-group-combobox.tsx) only
// offers whatever GET /api/user/self/groups returns, and that endpoint never
// included a user's personal BYOK group (service.BuildByokGroup) because it
// is per-user, not part of the admin-curated UserUsableGroups list. A
// customer who had saved a key still had no way to create a token that
// routed through it. GetUserGroups must add the group once — and only
// once — the user has actually configured a channel in it.
func TestGetUserGroupsExposesByokGroupOnlyOnceConfigured(t *testing.T) {
	user := setupByokControllerTest(t, 4202)
	group := service.BuildByokGroup(user.Id)

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/user/self/groups", nil, user.Id)
	GetUserGroups(ctx)
	before := decodeAPIResponse(t, recorder)
	require.True(t, before.Success, before.Message)

	var groupsBefore map[string]map[string]any
	require.NoError(t, common.Unmarshal(before.Data, &groupsBefore))
	_, present := groupsBefore[group]
	assert.False(t, present, "the BYOK group must not be selectable before any key is configured")

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(fmt.Sprintf(`{"default":1,%q:0}`, group)))
	require.NoError(t, model.DB.Create(&model.Channel{
		Type:  constant.ChannelTypeOpenAI,
		Key:   "sk-test",
		Group: group,
	}).Error)

	ctx2, recorder2 := newAuthenticatedContext(t, http.MethodGet, "/api/user/self/groups", nil, user.Id)
	GetUserGroups(ctx2)
	after := decodeAPIResponse(t, recorder2)
	require.True(t, after.Success, after.Message)

	var groupsAfter map[string]map[string]any
	require.NoError(t, common.Unmarshal(after.Data, &groupsAfter))
	entry, present := groupsAfter[group]
	require.True(t, present, "the BYOK group must become selectable once a channel is configured")
	assert.InDelta(t, 0, entry["ratio"], 0.0001, "BYOK usage must not carry a group-ratio markup")
	assert.Equal(t, "自带密钥（BYOK）", entry["desc"])
}

// TestDefaultByokModels protects a second bug found during BYOK deployment
// testing: SetByokKey (controller/byok.go) seeded a new BYOK channel's model
// list purely by copying model.GetFirstEnabledModelsForType(channelType) —
// i.e. by finding some other already-enabled, non-BYOK channel of the same
// provider and reusing its Models field. On a platform with no such
// admin-configured channel (or none enabled) for that provider, that lookup
// returns "", so the BYOK channel was created with an empty model list. An
// empty Models field means the channel can be routed to for no model at all
// (see model/ability.go's per-model Ability rows), so the customer's saved
// key silently did nothing: their token's group had no models to pick from.
// defaultByokModels must fill that gap from this project's own built-in
// per-provider catalog instead of leaving it empty.
func TestDefaultByokModels(t *testing.T) {
	t.Run("openai falls back to the built-in OpenAI catalog", func(t *testing.T) {
		got := defaultByokModels(constant.ChannelTypeOpenAI)
		require.NotEmpty(t, got, "an OpenAI BYOK channel must never end up with an empty model list")
		want := strings.Join(relay.GetAdaptor(constant.APITypeOpenAI).GetModelList(), ",")
		assert.Equal(t, want, got)
	})

	t.Run("anthropic falls back to the built-in Claude catalog", func(t *testing.T) {
		got := defaultByokModels(constant.ChannelTypeAnthropic)
		require.NotEmpty(t, got, "an Anthropic BYOK channel must never end up with an empty model list")
		want := strings.Join(relay.GetAdaptor(constant.APITypeAnthropic).GetModelList(), ",")
		assert.Equal(t, want, got)
	})

	t.Run("a channel type with no known API type mapping returns empty", func(t *testing.T) {
		assert.Empty(t, defaultByokModels(-1))
	})
}
