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
	"errors"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
)

// GetChannelByGroupAndType returns the single channel routed by group+type, or
// nil (with a nil error) when none exists yet. Self-service BYOK relies on this
// to decide whether a user's first key submission for a provider is an insert
// or an update, since each (user, provider) pair maps to exactly one channel.
func GetChannelByGroupAndType(group string, channelType int) (*Channel, error) {
	channel := &Channel{}
	err := DB.Where(commonGroupCol+" = ? and type = ?", group, channelType).First(channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return channel, nil
}

// UpsertByokChannel creates or updates the one channel that carries a single
// user's own upstream API key for one provider, scoped by group+type. It goes
// through Channel.Insert()/Channel.Update() (never a raw Create/Updates) so the
// abilities index that the relay router reads from stays in sync.
func UpsertByokChannel(group string, channelType int, name string, key string, models string) (*Channel, error) {
	existing, err := GetChannelByGroupAndType(group, channelType)
	if err != nil {
		return nil, err
	}

	// Route a BYOK channel through whatever base URL the platform's own
	// enabled channels for this provider already use, instead of leaving it
	// empty (which makes Channel.GetBaseURL() fall back to the provider's
	// public endpoint, see model/channel.go). On a deployment where the
	// server's own network route to that public endpoint is blocked (the
	// reason an internal tunnel/proxy base URL is configured on the
	// platform's own channels in the first place — see
	// ai-api-relay.md/byok-per-customer-channel-procedure.md), a BYOK
	// channel left on the public default fails outright with an upstream
	// "Country, region, or territory not supported" error even though the
	// customer's own key is perfectly valid: the request never has a chance
	// to reach the provider with a key that would work. This is purely a
	// routing setting, not a cost-driven restriction, so — unlike the model
	// list (see controller/byok.go's SetByokKey) — it is meant to be copied
	// uniformly onto every channel for this provider, BYOK included.
	baseURL := firstEnabledBaseURLForType(channelType)

	if existing != nil {
		existing.Key = key
		existing.Status = common.ChannelStatusEnabled
		if models != "" {
			existing.Models = models
		}
		// Re-saving a key is also the recovery path for a BYOK channel that
		// was created before this fix existed and is still stuck on the
		// public default: pick up the correct routing now rather than
		// requiring the customer to delete and recreate the channel.
		if baseURL != "" {
			existing.BaseURL = &baseURL
		}
		if err := existing.Update(); err != nil {
			return nil, err
		}
		return existing, nil
	}

	weight := uint(1)
	priority := int64(0)
	autoBan := 1
	channel := &Channel{
		Type:        channelType,
		Key:         key,
		Status:      common.ChannelStatusEnabled,
		Name:        name,
		Weight:      &weight,
		CreatedTime: common.GetTimestamp(),
		BaseURL:     &baseURL,
		Models:      models,
		Group:       group,
		Priority:    &priority,
		AutoBan:     &autoBan,
	}
	if err := channel.Insert(); err != nil {
		return nil, err
	}
	return channel, nil
}

// DeleteByokChannel removes the channel behind group+type, if one exists. It is
// a no-op (not an error) when the user never configured that provider.
func DeleteByokChannel(group string, channelType int) error {
	existing, err := GetChannelByGroupAndType(group, channelType)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	return existing.Delete()
}

// GetFirstEnabledModelsForType returns the model allow-list of the first
// enabled, non-BYOK channel of the given type. Note: as of the fix described
// in controller/byok.go's SetByokKey, this is no longer used to seed a new
// BYOK channel's model list (that always uses the full built-in catalog
// instead, to avoid propagating a deliberate, cost-driven admin restriction
// onto BYOK). It is kept here — unlike the model list, base URL routing
// (below) legitimately does need to copy from an existing channel, and
// having both lookups live in this file keeps the "copy from an existing
// channel of this type" query logic in one place — but nothing currently
// calls this function.
func GetFirstEnabledModelsForType(channelType int) string {
	var models string
	err := DB.Model(&Channel{}).
		Where("type = ? and status = ? and "+commonGroupCol+" NOT LIKE ?", channelType, common.ChannelStatusEnabled, "byok-u%").
		Order("id asc").
		Limit(1).
		Pluck("models", &models).Error
	if err != nil {
		return ""
	}
	return models
}

// firstEnabledBaseURLForType returns the BaseURL of the first enabled,
// non-BYOK channel of the given type, or "" if there isn't one (or it's
// blank). See the comment on UpsertByokChannel for why a BYOK channel's base
// URL is deliberately copied from an existing admin channel, unlike its
// model list.
func firstEnabledBaseURLForType(channelType int) string {
	var baseURL string
	err := DB.Model(&Channel{}).
		Where("type = ? and status = ? and "+commonGroupCol+" NOT LIKE ? and base_url IS NOT NULL and base_url != ''",
			channelType, common.ChannelStatusEnabled, "byok-u%").
		Order("id asc").
		Limit(1).
		Pluck("base_url", &baseURL).Error
	if err != nil {
		return ""
	}
	return baseURL
}
