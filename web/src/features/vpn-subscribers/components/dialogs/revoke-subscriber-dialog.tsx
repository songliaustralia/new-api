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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { handleServerError } from '@/lib/handle-server-error'

import { revokeVpnSubscription } from '../../api'
import { useVpnSubscribers } from '../vpn-subscribers-provider'

export function RevokeSubscriberDialog() {
  const { t } = useTranslation()
  const {
    subscriberDialog,
    setSubscriberDialog,
    currentSubscriber,
    triggerSubscribersRefresh,
  } = useVpnSubscribers()
  const [loading, setLoading] = useState(false)

  const open = subscriberDialog === 'revoke' && !!currentSubscriber
  if (!open || !currentSubscriber) return null

  const handleConfirm = async () => {
    setLoading(true)
    try {
      const res = await revokeVpnSubscription(currentSubscriber.user_id)
      if (res.success) {
        toast.success(t('Has been revoked'))
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
      title={t('Confirm revoke')}
      desc={t(
        'This immediately revokes VPN access for {{user}} and removes their client from the panel. Continue?',
        { user: currentSubscriber.username || `#${currentSubscriber.user_id}` }
      )}
      confirmText={t('Revoke')}
      handleConfirm={handleConfirm}
      isLoading={loading}
      destructive
    />
  )
}
