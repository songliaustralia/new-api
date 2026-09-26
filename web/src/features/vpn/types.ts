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

// ============================================================================
// API Response Envelope
// ============================================================================

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

// ============================================================================
// Public Plan
// ============================================================================

export interface VpnPlan {
  id: number
  title: string
  subtitle?: string
  price_amount: number
  currency: string
  duration_days: number
  // 0 means unlimited.
  total_gb: number
}

// ============================================================================
// Self Subscription Status
// ============================================================================

export interface VpnSelfStatus {
  has_subscription: boolean
  status?: string
  plan_id?: number
  plan_title?: string
  total_gb?: number
  // Unix seconds.
  start_time?: number
  end_time?: number
  active: boolean
  share_link?: string
  share_link_error?: string
}

// ============================================================================
// Payment
// ============================================================================

export interface VpnStripePayRequest {
  plan_id: number
}

export interface VpnStripePayResult {
  pay_link: string
}
