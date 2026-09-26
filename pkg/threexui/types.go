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

// Package threexui is a minimal REST client for a 3x-ui panel
// (github.com/MHSanaei/3x-ui), used to provision/renew/revoke VLESS clients
// for the VPN subscription feature. It only wraps the small slice of the
// panel's v3 API this feature needs (client add/update/delete/get and
// inbound listing), authenticated with a scoped Bearer API token created in
// the panel under Settings -> API Tokens (the "node-sync" scope is enough).
package threexui

// ClientConfig mirrors the subset of 3x-ui's own internal client model this
// integration needs. Field names/json tags must match 3x-ui's model.Client
// exactly, since they are marshaled straight into the panel's request body.
type ClientConfig struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Enable     bool   `json:"enable"`
	ExpiryTime int64  `json:"expiryTime"`      // unix millis, 0 = never expires
	TotalGB    int64  `json:"totalGB"`         // bytes, 0 = unlimited
	SubID      string `json:"subId,omitempty"` // 3x-ui subscription id
	Flow       string `json:"flow,omitempty"`  // e.g. "xtls-rprx-vision", empty is valid for plain Reality
	LimitIp    int    `json:"limitIp,omitempty"`
}

type addClientRequest struct {
	Client     ClientConfig `json:"client"`
	InboundIds []int        `json:"inboundIds"`
}

// apiResponse is the common envelope every 3x-ui panel API endpoint returns.
type apiResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Obj     any    `json:"obj,omitempty"`
}

// InboundSummary is a minimal projection of an inbound, used only by
// ListInbounds (the admin "test connection" action) to confirm a configured
// inbound id actually exists on the panel.
type InboundSummary struct {
	Id       int    `json:"id"`
	Remark   string `json:"remark"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Enable   bool   `json:"enable"`
}
