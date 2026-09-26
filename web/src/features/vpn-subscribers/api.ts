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
import { api } from '@/lib/api'

import type {
  ApiResponse,
  VpnPanelTestResult,
  VpnPlan,
  VpnPlanPayload,
  VpnSubscriber,
} from './types'

// ============================================================================
// Admin Plan Management
// ============================================================================

export async function getAdminVpnPlans(): Promise<ApiResponse<VpnPlan[]>> {
  const res = await api.get('/api/vpn/admin/plans')
  return res.data
}

export async function createVpnPlan(
  data: VpnPlanPayload
): Promise<ApiResponse<VpnPlan>> {
  const res = await api.post('/api/vpn/admin/plans', data)
  return res.data
}

export async function updateVpnPlan(
  id: number,
  data: VpnPlanPayload
): Promise<ApiResponse<VpnPlan>> {
  const res = await api.put(`/api/vpn/admin/plans/${id}`, data)
  return res.data
}

export async function patchVpnPlanStatus(
  id: number,
  enabled: boolean
): Promise<ApiResponse> {
  const res = await api.patch(`/api/vpn/admin/plans/${id}`, { enabled })
  return res.data
}

export async function deleteVpnPlan(id: number): Promise<ApiResponse> {
  const res = await api.delete(`/api/vpn/admin/plans/${id}`)
  return res.data
}

// ============================================================================
// Admin Subscriber Management
// ============================================================================

export async function getVpnSubscribers(): Promise<
  ApiResponse<VpnSubscriber[]>
> {
  const res = await api.get('/api/vpn/admin/subscribers')
  return res.data
}

export async function grantVpnSubscription(
  userId: number,
  planId: number
): Promise<ApiResponse> {
  const res = await api.post(`/api/vpn/admin/subscribers/${userId}/grant`, {
    plan_id: planId,
  })
  return res.data
}

export async function revokeVpnSubscription(
  userId: number
): Promise<ApiResponse> {
  const res = await api.post(`/api/vpn/admin/subscribers/${userId}/revoke`)
  return res.data
}

// ============================================================================
// Panel Connection Test
// ============================================================================

export async function testVpnPanelConnection(): Promise<
  ApiResponse<VpnPanelTestResult>
> {
  const res = await api.post('/api/vpn/admin/panel/test-connection')
  return res.data
}
