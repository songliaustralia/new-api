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
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var (
	ErrVpnOrderNotFound      = errors.New("vpn order not found")
	ErrVpnOrderStatusInvalid = errors.New("vpn order status invalid")
)

// VpnPlan is an admin-configured purchasable VPN period (e.g. "1 Month").
// Deliberately its own table rather than reusing SubscriptionPlan: a VPN
// period isn't a New API quota/group entitlement, it maps onto a 3x-ui panel
// client instead (see service.ProvisionOrRenewVpnClient).
type VpnPlan struct {
	Id int `json:"id"`

	Title    string `json:"title" gorm:"type:varchar(128);not null"`
	Subtitle string `json:"subtitle" gorm:"type:varchar(255);default:''"`

	PriceAmount float64 `json:"price_amount" gorm:"type:decimal(10,6);not null;default:0"`
	Currency    string  `json:"currency" gorm:"type:varchar(8);not null;default:'USD'"`

	// Access period length in days (e.g. 30 for "1 month").
	DurationDays int `json:"duration_days" gorm:"type:int;not null;default:30"`

	// Traffic cap for the period, in GB (0 = unlimited).
	TotalGB int `json:"total_gb" gorm:"type:int;not null;default:0"`

	Enabled   bool `json:"enabled" gorm:"default:true"`
	SortOrder int  `json:"sort_order" gorm:"type:int;default:0"`

	StripePriceId string `json:"stripe_price_id" gorm:"type:varchar(128);default:''"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (p *VpnPlan) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *VpnPlan) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

func (p *VpnPlan) Insert() error {
	return DB.Create(p).Error
}

func (p *VpnPlan) Update() error {
	return DB.Save(p).Error
}

func GetVpnPlanById(id int) (*VpnPlan, error) {
	if id <= 0 {
		return nil, errors.New("invalid plan id")
	}
	var plan VpnPlan
	if err := DB.Where("id = ?", id).First(&plan).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

func GetEnabledVpnPlans() ([]VpnPlan, error) {
	var plans []VpnPlan
	if err := DB.Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

func GetAllVpnPlans() ([]VpnPlan, error) {
	var plans []VpnPlan
	if err := DB.Order("sort_order asc, id asc").Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}

func DeleteVpnPlanById(id int) error {
	if id <= 0 {
		return errors.New("invalid plan id")
	}
	return DB.Where("id = ?", id).Delete(&VpnPlan{}).Error
}

// VpnOrder is one Stripe checkout attempt for a VpnPlan, mirroring
// SubscriptionOrder's shape and lifecycle (pending -> success/expired).
type VpnOrder struct {
	Id     int     `json:"id"`
	UserId int     `json:"user_id" gorm:"index"`
	PlanId int     `json:"plan_id" gorm:"index"`
	Money  float64 `json:"money"`

	TradeNo         string `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	Status          string `json:"status"`
	CreateTime      int64  `json:"create_time"`
	CompleteTime    int64  `json:"complete_time"`

	ProviderPayload string `json:"provider_payload" gorm:"type:text"`
}

func (o *VpnOrder) Insert() error {
	if o.CreateTime == 0 {
		o.CreateTime = common.GetTimestamp()
	}
	return DB.Create(o).Error
}

func GetVpnOrderByTradeNo(tradeNo string) *VpnOrder {
	if tradeNo == "" {
		return nil
	}
	var order VpnOrder
	if err := DB.Where("trade_no = ?", tradeNo).First(&order).Error; err != nil {
		return nil
	}
	return &order
}

func vpnTradeNoColumn() string {
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		return `"trade_no"`
	}
	return "`trade_no`"
}

// PrepareVpnOrderForCompletion validates a pending VPN order and returns it
// together with its plan, ready for the caller (service.CompleteVpnOrder) to
// provision panel access before calling FinalizeVpnOrder. Returns
// (nil, nil, nil) when the order was already completed previously (a Stripe
// webhook retry), which callers should treat as an idempotent no-op.
func PrepareVpnOrderForCompletion(tradeNo string, expectedPaymentProvider string) (*VpnOrder, *VpnPlan, error) {
	if tradeNo == "" {
		return nil, nil, errors.New("tradeNo is empty")
	}
	var order VpnOrder
	if err := DB.Where(vpnTradeNoColumn()+" = ?", tradeNo).First(&order).Error; err != nil {
		return nil, nil, ErrVpnOrderNotFound
	}
	if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
		return nil, nil, ErrPaymentMethodMismatch
	}
	if order.Status == common.TopUpStatusSuccess {
		return nil, nil, nil
	}
	if order.Status != common.TopUpStatusPending {
		return nil, nil, ErrVpnOrderStatusInvalid
	}
	plan, err := GetVpnPlanById(order.PlanId)
	if err != nil {
		return nil, nil, err
	}
	return &order, plan, nil
}

// FinalizeVpnOrder marks a VPN order paid after the panel client has already
// been provisioned/renewed successfully by the caller.
func FinalizeVpnOrder(order *VpnOrder, providerPayload string, planTitle string) error {
	if order == nil {
		return errors.New("order is nil")
	}
	order.Status = common.TopUpStatusSuccess
	order.CompleteTime = common.GetTimestamp()
	if providerPayload != "" {
		order.ProviderPayload = providerPayload
	}
	if err := DB.Save(order).Error; err != nil {
		return err
	}
	RecordLog(order.UserId, LogTypeTopup, fmt.Sprintf("VPN 订阅购买成功，套餐: %s，支付金额: %.2f", planTitle, order.Money))
	return nil
}

// ExpireVpnOrder marks a still-pending VPN order as expired (the user closed
// or abandoned the Stripe checkout page). No panel call is needed here since
// nothing was ever provisioned for a pending order.
func ExpireVpnOrder(tradeNo string, expectedPaymentProvider string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var order VpnOrder
		if err := lockForUpdate(tx).Where(vpnTradeNoColumn()+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrVpnOrderNotFound
		}
		if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if order.Status != common.TopUpStatusPending {
			return nil
		}
		order.Status = common.TopUpStatusExpired
		order.CompleteTime = common.GetTimestamp()
		return tx.Save(&order).Error
	})
}

func CountVpnOrdersByUserAndPlan(userId int, planId int, status string) (int64, error) {
	if userId <= 0 || planId <= 0 {
		return 0, errors.New("invalid userId or planId")
	}
	var count int64
	query := DB.Model(&VpnOrder{}).Where("user_id = ? AND plan_id = ?", userId, planId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// VpnSubscriber is the per-user VLESS client instance provisioned on the
// configured 3x-ui panel. One row per user (a repeat purchase renews/extends
// the same row and the same panel client rather than creating a second one).
type VpnSubscriber struct {
	Id     int `json:"id"`
	UserId int `json:"user_id" gorm:"uniqueIndex"`

	// ClientEmail is the identifier used as the 3x-ui client's "email" field
	// (its per-client unique key on the panel; unrelated to the user's real
	// email address). Derived from UserId alone, see service.BuildVpnClientEmail.
	ClientEmail string `json:"client_email" gorm:"type:varchar(128);unique"`
	ClientUUID  string `json:"client_uuid" gorm:"type:varchar(64)"`
	InboundId   int    `json:"inbound_id" gorm:"type:int"`
	SubId       string `json:"sub_id" gorm:"type:varchar(64)"`

	PlanId    int    `json:"plan_id" gorm:"index"`
	Status    string `json:"status" gorm:"type:varchar(32);index"` // active/revoked
	TotalGB   int    `json:"total_gb" gorm:"type:int;default:0"`
	StartTime int64  `json:"start_time" gorm:"bigint"`
	EndTime   int64  `json:"end_time" gorm:"bigint;index"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (s *VpnSubscriber) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

func (s *VpnSubscriber) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = common.GetTimestamp()
	return nil
}

func (s *VpnSubscriber) Insert() error {
	return DB.Create(s).Error
}

func (s *VpnSubscriber) Update() error {
	return DB.Save(s).Error
}

// GetVpnSubscriberByUserId returns (nil, nil) if the user has no subscriber
// row yet (never purchased), which callers should treat as a normal state,
// not an error.
func GetVpnSubscriberByUserId(userId int) (*VpnSubscriber, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	var sub VpnSubscriber
	err := DB.Where("user_id = ?", userId).First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func GetAllVpnSubscribers() ([]VpnSubscriber, error) {
	var subs []VpnSubscriber
	if err := DB.Order("end_time desc, id desc").Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}
