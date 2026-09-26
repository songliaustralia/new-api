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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Combobox } from '@/components/ui/combobox'
import { handleServerError } from '@/lib/handle-server-error'

import { getAdminVpnPlans, grantVpnSubscription } from '../../api'
import type { VpnPlan } from '../../types'
import { useVpnSubscribers } from '../vpn-subscribers-provider'

export function GrantSubscriberDialog() {
  const { t } = useTranslation()
  const {
    subscriberDialog,
    setSubscriberDialog,
    currentSubscriber,
    triggerSubscribersRefresh,
  } = useVpnSubscribers()
  const [plans, setPlans] = useState<VpnPlan[]>([])
  const [selectedPlanId, setSelectedPlanId] = useState('')
  const [loading, setLoading] = useState(false)

  const open = subscriberDialog === 'grant' && !!currentSubscriber

  useEffect(() => {
    if (!open) return
    setSelectedPlanId(
      currentSubscriber ? String(currentSubscriber.plan_id) : ''
    )
    getAdminVpnPlans()
      .then((res) => {
        if (res.success) setPlans(res.data || [])
      })
      .catch(() => {})
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  if (!open || !currentSubscriber) return null

  const handleConfirm = async () => {
    if (!selectedPlanId) {
      toast.error(t('Please select a subscription plan'))
      return
    }
    setLoading(true)
    try {
      const res = await grantVpnSubscription(
        currentSubscriber.user_id,
        Number(selectedPlanId)
      )
      if (res.success) {
        toast.success(t('Granted successfully'))
        triggerSubscribersRefresh()
        setSubscriberDialog(null)
      } else {
        handleServerError(res)
      }
    } catch (error) {
      handleServerError(error, t('Operation failed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <ConfirmDialog
      open
      onOpenChange={(v) => !v && setSubscriberDialog(null)}
      title={t('Grant/Extend VPN access')}
      desc={t(
        'Grant or extend VPN access for {{user}} with no payment required.',
        { user: currentSubscriber.username || `#${currentSubscriber.user_id}` }
      )}
      confirmText={t('Grant/Extend')}
      handleConfirm={handleConfirm}
      isLoading={loading}
      disabled={!selectedPlanId}
    >
      <Combobox
        options={plans.map((plan) => ({
          value: String(plan.id),
          label: `${plan.title} (${Number(plan.price_amount || 0).toFixed(2)} ${(plan.currency || 'usd').toUpperCase()})`,
        }))}
        value={selectedPlanId}
        onValueChange={(v) => v !== null && setSelectedPlanId(v)}
        className='w-full'
        placeholder={t('Select subscription plan')}
      />
    </ConfirmDialog>
  )
}
