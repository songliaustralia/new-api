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
// Plan Management
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
  enabled: boolean
  sort_order: number
  stripe_price_id?: string
  created_at?: number
  updated_at?: number
}

export interface VpnPlanPayload {
  title: string
  subtitle?: string
  price_amount: number
  currency: string
  duration_days: number
  total_gb: number
  enabled: boolean
  sort_order: number
  stripe_price_id?: string
}

// ============================================================================
// Subscriber Management
// ============================================================================

export interface VpnSubscriber {
  id: number
  user_id: number
  client_email: string
  client_uuid: string
  inbound_id: number
  sub_id: string
  plan_id: number
  status: string
  total_gb: number
  start_time: number
  end_time: number
  created_at: number
  updated_at: number
  username: string
  plan_title: string
  active: boolean
}

// ============================================================================
// Panel Connection Test
// ============================================================================

export interface VpnPanelTestResult {
  id: number
  remark: string
  port: number
  protocol: string
  enable: boolean
}

// ============================================================================
// Dialog Types
// ============================================================================

export type VpnPlanDialogType = 'create' | 'update' | 'toggle-status' | 'delete'

export type VpnSubscriberDialogType = 'grant' | 'revoke'
