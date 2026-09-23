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

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// byokProvider describes one BYOK-eligible upstream provider: the New API
// channel type it maps onto, and a minimal key-shape hint used only for a
// friendlier client-side error (never a security control by itself).
type byokProvider struct {
	channelType  int
	keyMinLength int
}

var byokProviders = map[string]byokProvider{
	"openai":    {channelType: constant.ChannelTypeOpenAI, keyMinLength: 20},
	"anthropic": {channelType: constant.ChannelTypeAnthropic, keyMinLength: 20},
}

type byokProviderStatus struct {
	Configured  bool   `json:"configured"`
	SetSince    int64  `json:"set_since,omitempty"`
	KeyMasked   string `json:"key_masked,omitempty"`
	ModelsCount int    `json:"models_count,omitempty"`
}

// byokStatusResponse tells the client, in one call, both what the user has
// already configured and whether they are currently allowed to configure
// anything at all. BYOK is a subscription perk (see SetByokKey), so a caller
// with no active subscription gets Eligible=false and an empty Providers map
// rather than a bare 403 the UI would have to special-case.
type byokStatusResponse struct {
	Eligible  bool                          `json:"eligible"`
	Providers map[string]byokProviderStatus `json:"providers"`
}

type byokSetRequest struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
}

// maskKey keeps only enough of a submitted key for the owner to recognize it
// in the UI later, never enough to reconstruct or reuse it.
func maskKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if len(trimmed) <= 8 {
		return "****"
	}
	return trimmed[:4] + "..." + trimmed[len(trimmed)-4:]
}

// GetByokStatus reports, for every supported provider, whether the current
// user has already configured their own key. It never returns the key itself.
// It also reports Eligible so the profile page can show a locked/upsell state
// for a user with no active subscription instead of an empty, confusing form.
func GetByokStatus(c *gin.Context) {
	userId := c.GetInt("id")

	eligible, err := model.HasActiveUserSubscription(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	group := service.BuildByokGroup(userId)

	statusByProvider := make(map[string]byokProviderStatus, len(byokProviders))
	for name, provider := range byokProviders {
		channel, err := model.GetChannelByGroupAndType(group, provider.channelType)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if channel == nil {
			statusByProvider[name] = byokProviderStatus{Configured: false}
			continue
		}
		modelsCount := 0
		if channel.Models != "" {
			modelsCount = len(strings.Split(channel.Models, ","))
		}
		statusByProvider[name] = byokProviderStatus{
			Configured:  true,
			SetSince:    channel.CreatedTime,
			KeyMasked:   maskKey(channel.Key),
			ModelsCount: modelsCount,
		}
	}
	common.ApiSuccess(c, byokStatusResponse{Eligible: eligible, Providers: statusByProvider})
}

// SetByokKey creates or replaces the current user's own key for one provider.
// It never touches any other user's data: the channel it writes to is always
// scoped to service.BuildByokGroup(userId), which is derived only from the
// authenticated caller's own id.
func SetByokKey(c *gin.Context) {
	userId := c.GetInt("id")

	// BYOK is a subscription perk, not a free-for-all: a request's own key
	// still routes for free (ratio 0, see below), so without this gate any
	// logged-in user could get unlimited AI usage through VocentraAI with no
	// revenue to VocentraAI at all. Requiring any currently-active
	// subscription (any plan tier) is the same check the wallet-overflow
	// billing path already uses (model.HasActiveUserSubscription).
	eligible, err := model.HasActiveUserSubscription(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !eligible {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该功能仅面向已订阅套餐的客户开放，请先订阅任意套餐后再设置",
		})
		return
	}

	var request byokSetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}

	provider, ok := byokProviders[request.Provider]
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("unsupported provider: %s", request.Provider),
		})
		return
	}

	key := strings.TrimSpace(request.Key)
	if len(key) < provider.keyMinLength {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该 Key 长度看起来不太对，请检查后重新粘贴",
		})
		return
	}
	if len(key) > 512 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Key 长度超出限制",
		})
		return
	}

	group := service.BuildByokGroup(userId)

	// A brand-new BYOK group needs a ratio entry before any token can route
	// through it (see service.IsUserOwnedByokGroup + ratio_setting.ContainsGroupRatio,
	// checked together at relay/token-selection time). Ratio 0 means the
	// platform deducts ZERO quota from the customer's own VocentraAI wallet
	// for requests in this group: the real AI cost is already being billed
	// directly to the customer's own OpenAI/Anthropic account through their
	// own key, so charging VocentraAI quota on top of that would be billing
	// them twice for the same tokens. VocentraAI monetizes BYOK through the
	// subscription gate above (SetByokKey requires an active subscription),
	// not through a per-token markup here.
	if !ratio_setting.ContainsGroupRatio(group) {
		ratio_setting.GetGroupRatioSetting().GroupRatio.Set(group, 0)
		if err := model.UpdateOption("GroupRatio", ratio_setting.GroupRatio2JSONString()); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	models := model.GetFirstEnabledModelsForType(provider.channelType)
	name := fmt.Sprintf("byok-u%d-%s", userId, request.Provider)
	if _, err := model.UpsertByokChannel(group, provider.channelType, name, key, models); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"provider": request.Provider})
}

// DeleteByokKey removes the current user's own key for one provider and stops
// routing their requests to it. It only ever operates on the caller's own
// group, for the same reason as SetByokKey above.
func DeleteByokKey(c *gin.Context) {
	userId := c.GetInt("id")
	providerName := c.Param("provider")
	provider, ok := byokProviders[providerName]
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("unsupported provider: %s", providerName),
		})
		return
	}

	group := service.BuildByokGroup(userId)
	if err := model.DeleteByokChannel(group, provider.channelType); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"provider": providerName})
}

// userHasAnyByokChannel reports whether the user has configured at least one
// BYOK provider channel. GetUserGroups (controller/group.go) uses this to
// decide whether the user's personal BYOK group should be exposed as a
// selectable token group at all: without a configured channel, offering the
// group would let a token be created that can never route anywhere.
func userHasAnyByokChannel(userId int) (bool, error) {
	group := service.BuildByokGroup(userId)
	for _, provider := range byokProviders {
		channel, err := model.GetChannelByGroupAndType(group, provider.channelType)
		if err != nil {
			return false, err
		}
		if channel != nil {
			return true, nil
		}
	}
	return false, nil
}
