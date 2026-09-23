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
package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupByokModelTest(t *testing.T) {
	t.Helper()
	truncateTables(t)
}

// TestUpsertByokChannelCopiesBaseURLFromAnExistingAdminChannel protects a
// fifth bug found during BYOK deployment testing: a newly created BYOK
// channel was left with an empty BaseURL, which Channel.GetBaseURL() then
// resolves to the provider's public default endpoint (see model/channel.go).
// On a deployment where the server's own network route to that public
// endpoint doesn't work — the reason an admin channel's BaseURL is pointed at
// an internal tunnel/proxy address in the first place — a BYOK channel left
// on the public default fails outright with an upstream error (observed:
// "Country, region, or territory not supported") even though the customer's
// own key is perfectly valid. Unlike the model list (see
// TestSetByokKeySeedsFullBuiltinCatalogEvenWhenAnAdminChannelIsRestricted in
// controller/byok_test.go), BaseURL is a purely technical routing setting
// with no cost implication, so it is supposed to be copied from whatever an
// existing enabled admin channel of the same provider type already uses.
func TestUpsertByokChannelCopiesBaseURLFromAnExistingAdminChannel(t *testing.T) {
	setupByokModelTest(t)

	adminBaseURL := "http://127.0.0.1:1082"
	require.NoError(t, DB.Create(&Channel{
		Type:    constant.ChannelTypeOpenAI,
		Key:     "sk-admin-main",
		Status:  common.ChannelStatusEnabled,
		Group:   "default",
		Models:  "gpt-4o",
		BaseURL: &adminBaseURL,
	}).Error)

	channel, err := UpsertByokChannel("byok-u9001", constant.ChannelTypeOpenAI, "byok-u9001-openai", "sk-customer-key", "gpt-4o,gpt-4o-mini")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.NotNil(t, channel.BaseURL, "a new BYOK channel must not be left with a nil BaseURL when an admin channel's routing is available to copy")
	assert.Equal(t, adminBaseURL, *channel.BaseURL, "a new BYOK channel must route through the same base URL the platform's own channel for this provider uses")

	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	require.NotNil(t, stored.BaseURL)
	assert.Equal(t, adminBaseURL, *stored.BaseURL, "the copied base URL must actually be persisted, not just held on the in-memory struct")
}

// TestUpsertByokChannelHealsAnExistingBlankBaseURLOnResave protects the
// recovery path for a BYOK channel that was created before the fix above
// existed (or before any admin channel with a usable BaseURL existed) and is
// still stuck on the public default: re-saving the same key must pick up the
// correct routing, not just refresh the key and model list.
func TestUpsertByokChannelHealsAnExistingBlankBaseURLOnResave(t *testing.T) {
	setupByokModelTest(t)

	emptyBaseURL := ""
	broken := &Channel{
		Type:    constant.ChannelTypeOpenAI,
		Key:     "sk-customer-key-old",
		Status:  common.ChannelStatusEnabled,
		Group:   "byok-u9002",
		Models:  "gpt-4o",
		BaseURL: &emptyBaseURL,
	}
	require.NoError(t, broken.Insert())

	adminBaseURL := "http://127.0.0.1:1082"
	require.NoError(t, DB.Create(&Channel{
		Type:    constant.ChannelTypeOpenAI,
		Key:     "sk-admin-main",
		Status:  common.ChannelStatusEnabled,
		Group:   "default",
		Models:  "gpt-4o",
		BaseURL: &adminBaseURL,
	}).Error)

	channel, err := UpsertByokChannel("byok-u9002", constant.ChannelTypeOpenAI, "byok-u9002-openai", "sk-customer-key-new", "gpt-4o,gpt-4o-mini")
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.NotNil(t, channel.BaseURL)
	assert.Equal(t, adminBaseURL, *channel.BaseURL, "re-saving a key must heal an existing channel that was stuck on an empty/public-default BaseURL")
}

// TestFirstEnabledBaseURLForTypeIgnoresByokAndBlankChannels protects the
// lookup UpsertByokChannel relies on: it must skip other BYOK channels
// (whose own BaseURL may itself still be wrong) and channels with no BaseURL
// configured, and find the first enabled admin channel that actually has one.
func TestFirstEnabledBaseURLForTypeIgnoresByokAndBlankChannels(t *testing.T) {
	setupByokModelTest(t)

	assert.Empty(t, firstEnabledBaseURLForType(constant.ChannelTypeOpenAI), "no channels configured yet must yield no base URL to copy")

	emptyBaseURL := ""
	require.NoError(t, DB.Create(&Channel{
		Type:    constant.ChannelTypeOpenAI,
		Key:     "sk-no-routing",
		Status:  common.ChannelStatusEnabled,
		Group:   "default",
		BaseURL: &emptyBaseURL,
	}).Error)
	assert.Empty(t, firstEnabledBaseURLForType(constant.ChannelTypeOpenAI), "a channel with a blank BaseURL must not be copied")

	otherByokBaseURL := "http://127.0.0.1:9999"
	require.NoError(t, DB.Create(&Channel{
		Type:    constant.ChannelTypeOpenAI,
		Key:     "sk-other-byok-customer",
		Status:  common.ChannelStatusEnabled,
		Group:   "byok-u1",
		BaseURL: &otherByokBaseURL,
	}).Error)
	assert.Empty(t, firstEnabledBaseURLForType(constant.ChannelTypeOpenAI), "another customer's own BYOK channel must never be copied onto a new one")

	adminBaseURL := "http://127.0.0.1:1082"
	require.NoError(t, DB.Create(&Channel{
		Type:    constant.ChannelTypeOpenAI,
		Key:     "sk-admin-main",
		Status:  common.ChannelStatusEnabled,
		Group:   "default",
		BaseURL: &adminBaseURL,
	}).Error)
	assert.Equal(t, adminBaseURL, firstEnabledBaseURLForType(constant.ChannelTypeOpenAI))
}
