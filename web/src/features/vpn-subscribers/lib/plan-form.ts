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
import type { TFunction } from 'i18next'
import { z } from 'zod'

import type { VpnPlan, VpnPlanPayload } from '../types'

export function getVpnPlanFormSchema(t: TFunction) {
  return z.object({
    title: z.string().min(1, t('Please enter plan title')),
    subtitle: z.string().optional(),
    price_amount: z.coerce.number().min(0, t('Please enter amount')),
    currency: z.string().min(1, t('Please enter currency')),
    duration_days: z.coerce.number().min(1),
    total_gb: z.coerce.number().min(0),
    enabled: z.boolean(),
    sort_order: z.coerce.number(),
    stripe_price_id: z.string().optional(),
  })
}

export type VpnPlanFormValues = z.infer<ReturnType<typeof getVpnPlanFormSchema>>

export const VPN_PLAN_FORM_DEFAULTS: VpnPlanFormValues = {
  title: '',
  subtitle: '',
  price_amount: 0,
  currency: 'usd',
  duration_days: 30,
  total_gb: 0,
  enabled: true,
  sort_order: 0,
  stripe_price_id: '',
}

export function vpnPlanToFormValues(plan: VpnPlan): VpnPlanFormValues {
  return {
    title: plan.title || '',
    subtitle: plan.subtitle || '',
    price_amount: Number(plan.price_amount || 0),
    currency: plan.currency || 'usd',
    duration_days: Number(plan.duration_days || 1),
    total_gb: Number(plan.total_gb || 0),
    enabled: plan.enabled !== false,
    sort_order: Number(plan.sort_order || 0),
    stripe_price_id: plan.stripe_price_id || '',
  }
}

export function vpnPlanFormValuesToPayload(
  values: VpnPlanFormValues
): VpnPlanPayload {
  return {
    title: values.title,
    subtitle: values.subtitle || '',
    price_amount: Number(values.price_amount || 0),
    currency: values.currency || 'usd',
    duration_days: Number(values.duration_days || 0),
    total_gb: Number(values.total_gb || 0),
    enabled: values.enabled,
    sort_order: Number(values.sort_order || 0),
    stripe_price_id: values.stripe_price_id || '',
  }
}
