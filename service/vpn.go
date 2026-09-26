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
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/threexui"

	"github.com/google/uuid"
)

const vpnClientEmailPrefix = "vpn-u"

// BuildVpnClientEmail returns the identifier used as the 3x-ui client's
// "email" field for a given user, derived from userId alone (the same
// "namespace string derived from an id, never a separate ownership table"
// approach BuildByokGroup uses).
func BuildVpnClientEmail(userId int) string {
	return fmt.Sprintf("%s%d", vpnClientEmailPrefix, userId)
}

// vpnOption reads one VPN-related admin setting from the shared option map.
func vpnOption(key string) string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return strings.TrimSpace(common.OptionMap[key])
}

func vpnPanelClient() (client *threexui.Client, inboundId int, err error) {
	baseUrl := vpnOption("VpnPanelBaseUrl")
	token := vpnOption("VpnPanelApiToken")
	inboundIdStr := vpnOption("VpnPanelInboundId")
	inboundId, _ = strconv.Atoi(inboundIdStr)
	if baseUrl == "" || token == "" || inboundId <= 0 {
		return nil, 0, fmt.Errorf("VPN 面板尚未在后台配置完整（面板地址 / API 令牌 / Inbound ID）")
	}
	return threexui.NewClient(baseUrl, token), inboundId, nil
}

// ProvisionOrRenewVpnClient creates (first purchase) or extends (renewal)
// the user's single VLESS client on the configured 3x-ui inbound, and
// upserts the matching model.VpnSubscriber row to reflect it. Renewing
// before the current access has expired extends from the existing expiry
// rather than from "now", so an early renewal never shortens remaining
// access. Also used by the admin "grant/extend" action (a manual, unpaid
// provision), not only by paid orders.
func ProvisionOrRenewVpnClient(userId int, plan *model.VpnPlan) (*model.VpnSubscriber, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan is nil")
	}
	if userId <= 0 {
		return nil, fmt.Errorf("invalid userId")
	}

	client, inboundId, err := vpnPanelClient()
	if err != nil {
		return nil, err
	}

	existing, err := model.GetVpnSubscriberByUserId(userId)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	base := now
	if existing != nil && existing.Status == "active" && existing.EndTime > now.Unix() {
		base = time.Unix(existing.EndTime, 0)
	}
	endTime := base.AddDate(0, 0, plan.DurationDays)
	expiryMillis := endTime.UnixMilli()

	var totalGBBytes int64
	if plan.TotalGB > 0 {
		totalGBBytes = int64(plan.TotalGB) * 1024 * 1024 * 1024
	}

	email := BuildVpnClientEmail(userId)
	flow := vpnOption("VpnRealityFlow")

	if existing == nil {
		clientUUID := uuid.NewString()
		subId := strings.ReplaceAll(uuid.NewString(), "-", "")
		cfg := threexui.ClientConfig{
			ID:         clientUUID,
			Email:      email,
			Enable:     true,
			ExpiryTime: expiryMillis,
			TotalGB:    totalGBBytes,
			SubID:      subId,
			Flow:       flow,
		}
		if err := client.AddClient(inboundId, cfg); err != nil {
			return nil, fmt.Errorf("3x-ui 创建客户端失败: %w", err)
		}
		sub := &model.VpnSubscriber{
			UserId:      userId,
			ClientEmail: email,
			ClientUUID:  clientUUID,
			InboundId:   inboundId,
			SubId:       subId,
			PlanId:      plan.Id,
			Status:      "active",
			TotalGB:     plan.TotalGB,
			StartTime:   now.Unix(),
			EndTime:     endTime.Unix(),
		}
		if err := sub.Insert(); err != nil {
			return nil, err
		}
		return sub, nil
	}

	cfg := threexui.ClientConfig{
		ID:         existing.ClientUUID,
		Email:      email,
		Enable:     true,
		ExpiryTime: expiryMillis,
		TotalGB:    totalGBBytes,
		SubID:      existing.SubId,
		Flow:       flow,
	}
	if err := client.UpdateClient(email, inboundId, cfg); err != nil {
		return nil, fmt.Errorf("3x-ui 续期客户端失败: %w", err)
	}
	existing.PlanId = plan.Id
	existing.Status = "active"
	existing.TotalGB = plan.TotalGB
	existing.EndTime = endTime.Unix()
	existing.InboundId = inboundId
	if existing.StartTime == 0 {
		existing.StartTime = now.Unix()
	}
	if err := existing.Update(); err != nil {
		return nil, err
	}
	return existing, nil
}

// RevokeVpnClient disables the user's VLESS client on the panel (rather than
// deleting it outright) and marks the subscriber row revoked, so a future
// admin grant or paid renewal can simply re-enable the same UUID.
func RevokeVpnClient(userId int) error {
	sub, err := model.GetVpnSubscriberByUserId(userId)
	if err != nil {
		return err
	}
	if sub == nil {
		return fmt.Errorf("该用户没有 VPN 订阅记录")
	}
	client, inboundId, err := vpnPanelClient()
	if err != nil {
		return err
	}
	cfg := threexui.ClientConfig{
		ID:         sub.ClientUUID,
		Email:      sub.ClientEmail,
		Enable:     false,
		ExpiryTime: sub.EndTime * 1000,
		SubID:      sub.SubId,
		Flow:       vpnOption("VpnRealityFlow"),
	}
	if err := client.UpdateClient(sub.ClientEmail, inboundId, cfg); err != nil {
		return fmt.Errorf("3x-ui 禁用客户端失败: %w", err)
	}
	sub.Status = "revoked"
	sub.EndTime = common.GetTimestamp()
	return sub.Update()
}

// TestVpnPanelConnection confirms the configured base URL/token/inbound id
// actually work together, for the admin settings page's "test connection"
// button, without touching any client.
func TestVpnPanelConnection() (*threexui.InboundSummary, error) {
	client, inboundId, err := vpnPanelClient()
	if err != nil {
		return nil, err
	}
	inbounds, err := client.ListInbounds()
	if err != nil {
		return nil, err
	}
	for _, ib := range inbounds {
		if ib.Id == inboundId {
			return &ib, nil
		}
	}
	return nil, fmt.Errorf("面板连接成功，但未找到 Inbound ID = %d（共 %d 个 inbound）", inboundId, len(inbounds))
}

// BuildVpnShareLink constructs a ready-to-import vless:// URI for the user's
// current client, from the Reality parameters an admin enters once in the
// VPN panel settings form. 3x-ui's own subscription HTTP server (a separate
// port/path) is deliberately not used, so this feature needs no extra port
// opened on the VPS beyond the VLESS inbound itself.
func BuildVpnShareLink(sub *model.VpnSubscriber) (string, error) {
	if sub == nil {
		return "", fmt.Errorf("subscriber is nil")
	}
	host := vpnOption("VpnServerHost")
	port := vpnOption("VpnServerPort")
	network := vpnOption("VpnNetwork")
	pbk := vpnOption("VpnRealityPublicKey")
	sni := vpnOption("VpnRealitySni")
	sid := vpnOption("VpnRealityShortId")
	spx := vpnOption("VpnRealitySpiderX")
	fp := vpnOption("VpnRealityFingerprint")
	flow := vpnOption("VpnRealityFlow")

	if host == "" || port == "" || pbk == "" || sni == "" {
		return "", fmt.Errorf("VPN 服务器 / Reality 参数尚未在后台配置完整")
	}
	if network == "" {
		network = "tcp"
	}
	if fp == "" {
		fp = "chrome"
	}

	q := url.Values{}
	q.Set("type", network)
	q.Set("security", "reality")
	q.Set("pbk", pbk)
	q.Set("fp", fp)
	q.Set("sni", sni)
	if sid != "" {
		q.Set("sid", sid)
	}
	if spx != "" {
		q.Set("spx", spx)
	}
	if flow != "" {
		q.Set("flow", flow)
	}

	return fmt.Sprintf("vless://%s@%s:%s?%s#%s",
		sub.ClientUUID, host, port, q.Encode(), url.PathEscape("VocentraAI-VPN")), nil
}

// CompleteVpnOrder mirrors model.CompleteSubscriptionOrder's contract exactly
// (same not-found/idempotent/status-invalid semantics), so
// controller/topup_stripe.go's fulfillOrder can try it the same way it tries
// CompleteSubscriptionOrder. It lives in the service package rather than
// model because it must call out to the 3x-ui panel via
// ProvisionOrRenewVpnClient, and model must never import service.
//
// Concurrency note: unlike CompleteSubscriptionOrder, this does not wrap the
// whole operation in a DB row-locking transaction. The Stripe webhook path
// that calls this already serializes retries for the same trade number via
// controller.LockOrder/UnlockOrder (an in-process mutex keyed by trade_no),
// which is enough for this feature's single-instance, friends-scale
// deployment; it would not be enough for a multi-instance deployment, where
// the subscription feature's heavier transaction+row-lock pattern would be
// needed instead.
func CompleteVpnOrder(tradeNo string, providerPayload string, expectedPaymentProvider string) error {
	order, plan, err := model.PrepareVpnOrderForCompletion(tradeNo, expectedPaymentProvider)
	if err != nil {
		return err
	}
	if order == nil {
		return nil // already completed by an earlier delivery of the same webhook event
	}
	if _, err := ProvisionOrRenewVpnClient(order.UserId, plan); err != nil {
		return fmt.Errorf("provision vpn client failed: %w", err)
	}
	return model.FinalizeVpnOrder(order, providerPayload, plan.Title)
}
