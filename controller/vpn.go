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
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/thanhpk/randstr"
)

// ---- Self-service ----

// vpnPlanPublic is the subset of VpnPlan shown to end users (never exposes
// StripePriceId).
type vpnPlanPublic struct {
	Id           int     `json:"id"`
	Title        string  `json:"title"`
	Subtitle     string  `json:"subtitle"`
	PriceAmount  float64 `json:"price_amount"`
	Currency     string  `json:"currency"`
	DurationDays int     `json:"duration_days"`
	TotalGB      int     `json:"total_gb"`
}

func toVpnPlanPublic(p model.VpnPlan) vpnPlanPublic {
	return vpnPlanPublic{
		Id:           p.Id,
		Title:        p.Title,
		Subtitle:     p.Subtitle,
		PriceAmount:  p.PriceAmount,
		Currency:     p.Currency,
		DurationDays: p.DurationDays,
		TotalGB:      p.TotalGB,
	}
}

// GetVpnPlans lists the plans currently open for purchase.
func GetVpnPlans(c *gin.Context) {
	plans, err := model.GetEnabledVpnPlans()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	result := make([]vpnPlanPublic, 0, len(plans))
	for _, p := range plans {
		result = append(result, toVpnPlanPublic(p))
	}
	common.ApiSuccess(c, result)
}

type vpnSelfResponse struct {
	HasSubscription bool   `json:"has_subscription"`
	Status          string `json:"status,omitempty"`
	PlanId          int    `json:"plan_id,omitempty"`
	PlanTitle       string `json:"plan_title,omitempty"`
	TotalGB         int    `json:"total_gb,omitempty"`
	StartTime       int64  `json:"start_time,omitempty"`
	EndTime         int64  `json:"end_time,omitempty"`
	Active          bool   `json:"active"`
	ShareLink       string `json:"share_link,omitempty"`
	ShareLinkError  string `json:"share_link_error,omitempty"`
}

// GetVpnSelf returns the current user's VPN subscriber status, including a
// ready-to-import share link when their access is currently active.
func GetVpnSelf(c *gin.Context) {
	userId := c.GetInt("id")
	sub, err := model.GetVpnSubscriberByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if sub == nil {
		common.ApiSuccess(c, vpnSelfResponse{HasSubscription: false})
		return
	}

	resp := vpnSelfResponse{
		HasSubscription: true,
		Status:          sub.Status,
		PlanId:          sub.PlanId,
		TotalGB:         sub.TotalGB,
		StartTime:       sub.StartTime,
		EndTime:         sub.EndTime,
		Active:          sub.Status == "active" && sub.EndTime > time.Now().Unix(),
	}
	if plan, err := model.GetVpnPlanById(sub.PlanId); err == nil && plan != nil {
		resp.PlanTitle = plan.Title
	}
	if resp.Active {
		link, err := service.BuildVpnShareLink(sub)
		if err != nil {
			resp.ShareLinkError = err.Error()
		} else {
			resp.ShareLink = link
		}
	}
	common.ApiSuccess(c, resp)
}

type vpnStripePayRequest struct {
	PlanId int `json:"plan_id"`
}

// VpnRequestStripePay creates a one-time Stripe Checkout session for a VPN
// plan. Deliberately Checkout "payment" mode (not "subscription" mode, unlike
// the platform's own subscription plans): the friend simply buys another
// period whenever theirs is running out, which avoids taking on Stripe
// recurring-invoice webhook handling for what is a small, casual, few-friends
// feature. See CompleteVpnOrder / ProvisionOrRenewVpnClient for the renewal
// (extend-from-current-expiry) behavior this implies.
func VpnRequestStripePay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}
	if strings.TrimSpace(vpnOptionForController("VpnFeatureEnabled")) != "true" {
		common.ApiErrorMsg(c, "VPN 功能当前未开放")
		return
	}

	var req vpnStripePayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	plan, err := model.GetVpnPlanById(req.PlanId)
	if err != nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	if !plan.Enabled {
		common.ApiErrorMsg(c, "套餐未启用")
		return
	}
	if plan.StripePriceId == "" {
		common.ApiErrorMsg(c, "该套餐未配置 Stripe Price ID")
		return
	}
	if !strings.HasPrefix(setting.StripeApiSecret, "sk_") && !strings.HasPrefix(setting.StripeApiSecret, "rk_") {
		common.ApiErrorMsg(c, "Stripe 未配置或密钥无效")
		return
	}
	if setting.StripeWebhookSecret == "" {
		common.ApiErrorMsg(c, "Stripe Webhook 未配置")
		return
	}

	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil || user == nil {
		common.ApiErrorMsg(c, "用户不存在")
		return
	}

	reference := fmt.Sprintf("vpn-stripe-ref-%d-%d-%s", user.Id, time.Now().UnixMilli(), randstr.String(4))
	referenceId := "vpn_ref_" + common.Sha1([]byte(reference))

	payLink, err := genVpnStripeLink(referenceId, user.StripeCustomer, user.Email, plan.StripePriceId)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Stripe 创建 VPN Checkout Session 失败 user_id=%d trade_no=%s plan_id=%d error=%q", userId, referenceId, plan.Id, err.Error()))
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}

	order := &model.VpnOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           plan.PriceAmount,
		TradeNo:         referenceId,
		PaymentMethod:   model.PaymentMethodStripe,
		PaymentProvider: model.PaymentProviderStripe,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := order.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("创建 VPN 订单失败 user_id=%d trade_no=%s error=%q", userId, referenceId, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}

	common.ApiSuccess(c, gin.H{"pay_link": payLink})
}

// genVpnStripeLink is a one-time-payment sibling of subscription_payment_stripe.go's
// genStripeSubscriptionLink (which uses Checkout "subscription" mode instead).
func genVpnStripeLink(referenceId string, customerId string, email string, priceId string) (string, error) {
	stripe.Key = setting.StripeApiSecret
	params := &stripe.CheckoutSessionParams{
		ClientReferenceID: stripe.String(referenceId),
		SuccessURL:        stripe.String(paymentReturnPath("/vpn")),
		CancelURL:         stripe.String(paymentReturnPath("/vpn")),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceId),
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
	}
	if "" == customerId {
		if "" != email {
			params.CustomerEmail = stripe.String(email)
		}
		params.CustomerCreation = stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways))
	} else {
		params.Customer = stripe.String(customerId)
	}
	result, err := session.New(params)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func vpnOptionForController(key string) string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return common.OptionMap[key]
}

// ---- Admin: plan management ----

func AdminListVpnPlans(c *gin.Context) {
	plans, err := model.GetAllVpnPlans()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plans)
}

type vpnPlanUpsertRequest struct {
	Title         string  `json:"title"`
	Subtitle      string  `json:"subtitle"`
	PriceAmount   float64 `json:"price_amount"`
	Currency      string  `json:"currency"`
	DurationDays  int     `json:"duration_days"`
	TotalGB       int     `json:"total_gb"`
	Enabled       *bool   `json:"enabled"`
	SortOrder     int     `json:"sort_order"`
	StripePriceId string  `json:"stripe_price_id"`
}

func AdminCreateVpnPlan(c *gin.Context) {
	var req vpnPlanUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		common.ApiErrorMsg(c, "套餐名称不能为空")
		return
	}
	if req.DurationDays <= 0 {
		common.ApiErrorMsg(c, "有效期天数必须大于 0")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "USD"
	}
	plan := &model.VpnPlan{
		Title:         strings.TrimSpace(req.Title),
		Subtitle:      req.Subtitle,
		PriceAmount:   req.PriceAmount,
		Currency:      currency,
		DurationDays:  req.DurationDays,
		TotalGB:       req.TotalGB,
		Enabled:       enabled,
		SortOrder:     req.SortOrder,
		StripePriceId: strings.TrimSpace(req.StripePriceId),
	}
	if err := plan.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func AdminUpdateVpnPlan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的套餐 ID")
		return
	}
	plan, err := model.GetVpnPlanById(id)
	if err != nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	var req vpnPlanUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		common.ApiErrorMsg(c, "套餐名称不能为空")
		return
	}
	if req.DurationDays <= 0 {
		common.ApiErrorMsg(c, "有效期天数必须大于 0")
		return
	}
	plan.Title = strings.TrimSpace(req.Title)
	plan.Subtitle = req.Subtitle
	plan.PriceAmount = req.PriceAmount
	if strings.TrimSpace(req.Currency) != "" {
		plan.Currency = strings.TrimSpace(req.Currency)
	}
	plan.DurationDays = req.DurationDays
	plan.TotalGB = req.TotalGB
	if req.Enabled != nil {
		plan.Enabled = *req.Enabled
	}
	plan.SortOrder = req.SortOrder
	plan.StripePriceId = strings.TrimSpace(req.StripePriceId)
	if err := plan.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func AdminUpdateVpnPlanStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的套餐 ID")
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan, err := model.GetVpnPlanById(id)
	if err != nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	plan.Enabled = req.Enabled
	if err := plan.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func AdminDeleteVpnPlan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的套餐 ID")
		return
	}
	if err := model.DeleteVpnPlanById(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- Admin: subscriber management ----

type vpnSubscriberAdminView struct {
	model.VpnSubscriber
	Username  string `json:"username"`
	PlanTitle string `json:"plan_title"`
	Active    bool   `json:"active"`
}

func AdminListVpnSubscribers(c *gin.Context) {
	subs, err := model.GetAllVpnSubscribers()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	now := time.Now().Unix()
	planTitles := make(map[int]string)
	result := make([]vpnSubscriberAdminView, 0, len(subs))
	for _, sub := range subs {
		username, _ := model.GetUsernameById(sub.UserId, false)
		title, ok := planTitles[sub.PlanId]
		if !ok {
			if plan, err := model.GetVpnPlanById(sub.PlanId); err == nil && plan != nil {
				title = plan.Title
			}
			planTitles[sub.PlanId] = title
		}
		result = append(result, vpnSubscriberAdminView{
			VpnSubscriber: sub,
			Username:      username,
			PlanTitle:     title,
			Active:        sub.Status == "active" && sub.EndTime > now,
		})
	}
	common.ApiSuccess(c, result)
}

type vpnGrantRequest struct {
	PlanId int `json:"plan_id"`
}

// AdminGrantVpnSubscriber manually provisions or extends a user's VPN access
// with no payment involved (e.g. giving a friend free access, or fixing up a
// subscriber by hand). Reuses the exact same provisioning path a paid order
// uses.
func AdminGrantVpnSubscriber(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}
	var req vpnGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan, err := model.GetVpnPlanById(req.PlanId)
	if err != nil {
		common.ApiErrorMsg(c, "套餐不存在")
		return
	}
	if _, err := model.GetUserById(userId, false); err != nil {
		common.ApiErrorMsg(c, "用户不存在")
		return
	}
	sub, err := service.ProvisionOrRenewVpnClient(userId, plan)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.RecordLog(userId, model.LogTypeTopup, fmt.Sprintf("管理员手动开通/续期 VPN，套餐: %s", plan.Title))
	common.ApiSuccess(c, sub)
}

func AdminRevokeVpnSubscriber(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userId <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return
	}
	if err := service.RevokeVpnClient(userId); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RecordLog(userId, model.LogTypeTopup, "管理员吊销了 VPN 订阅")
	common.ApiSuccess(c, nil)
}

// AdminTestVpnPanelConnection lets the admin verify the panel settings form
// (base URL / API token / inbound id) before relying on it for real orders.
func AdminTestVpnPanelConnection(c *gin.Context) {
	inbound, err := service.TestVpnPanelConnection()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, inbound)
}
