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
	"github.com/QuantumNous/new-api/relay"
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

	// Always seed a BYOK channel from this project's own built-in model
	// catalog for the provider, never by copying an admin-configured
	// channel's (possibly deliberately restricted) model list.
	//
	// An earlier version of this code preferred copying from
	// model.GetFirstEnabledModelsForType(provider.channelType) — the first
	// enabled, non-BYOK channel of the same type — on the theory that this
	// keeps BYOK "in step with whatever the platform already curates". In
	// practice this backfired: an admin may restrict a channel's model list
	// for cost-control reasons that only make sense when the platform's own
	// API key is paying the bill (e.g. gating an expensive flagship model to
	// higher subscription tiers so a low-tier customer can't drain the
	// shared quota pool with a few requests — see
	// subscription-package-plan-design.md's "模型访问限制" section). BYOK
	// channels don't share that cost exposure at all: the customer's own
	// upstream key pays for every request, and the BYOK group's ratio is 0,
	// so VocentraAI's quota pool is never touched regardless of which model
	// is called. Copying a cost-driven restriction onto a BYOK channel just
	// hides models a customer has every right to call through their own key,
	// with no offsetting benefit. So BYOK always gets the full built-in
	// catalog instead.
	models := defaultByokModels(provider.channelType)
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

// defaultByokModels returns this project's own built-in model catalog for a
// provider — the same per-channel-type list relay.GetAdaptor(...).GetModelList()
// feeds into the public /v1/models endpoint, see controller/model.go's
// init(). SetByokKey uses this, unconditionally, to seed every BYOK
// channel's model list (see the comment there for why it never copies from
// an admin-configured channel instead). Returns "" if the channel type has
// no known API type mapping (should not happen for the providers in
// byokProviders).
func defaultByokModels(channelType int) string {
	apiType, ok := common.ChannelType2APIType(channelType)
	if !ok {
		return ""
	}
	return strings.Join(relay.GetAdaptor(apiType).GetModelList(), ",")
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

// userOwnedByokGroup returns the current user's personal BYOK group name
// together with whether they have actually configured a provider channel in
// it. The BYOK group (service.BuildByokGroup) is per-user and deliberately
// never appears in service.GetUserUsableGroups' admin-curated map (see the
// comment on GetUserGroups in controller/group.go), so any endpoint that
// gates a requested group name against that map has to recognize a
// configured BYOK customer's own group separately, or it looks unrecognized
// to the very account it belongs to. GetUserGroups and GetUserModels
// (controller/user.go) both need this same check.
func userOwnedByokGroup(userId int) (string, bool, error) {
	has, err := userHasAnyByokChannel(userId)
	if err != nil {
		return "", false, err
	}
	if !has {
		return "", false, nil
	}
	return service.BuildByokGroup(userId), true, nil
}
