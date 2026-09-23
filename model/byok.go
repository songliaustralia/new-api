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
	if existing != nil {
		existing.Key = key
		existing.Status = common.ChannelStatusEnabled
		if models != "" {
			existing.Models = models
		}
		if err := existing.Update(); err != nil {
			return nil, err
		}
		return existing, nil
	}

	weight := uint(1)
	priority := int64(0)
	autoBan := 1
	emptyBaseURL := ""
	channel := &Channel{
		Type:        channelType,
		Key:         key,
		Status:      common.ChannelStatusEnabled,
		Name:        name,
		Weight:      &weight,
		CreatedTime: common.GetTimestamp(),
		BaseURL:     &emptyBaseURL,
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
// enabled, non-BYOK channel of the given type. A new BYOK channel is seeded
// with the same models the platform's own pooled channel already exposes for
// that provider, instead of a hardcoded list here that would drift out of date
// as new models ship.
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
