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
import { useQuery } from '@tanstack/react-query'
import { ShieldCheck, ShieldOff, Wifi } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { StatusBadge } from '@/components/status-badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import { createServerError } from '@/lib/server-error-message'

import { getVpnSelf } from '../api'
import { formatVpnDataCap, formatVpnTimestamp } from '../lib'
import type { VpnSelfStatus } from '../types'

const EMPTY_VPN_STATUS: VpnSelfStatus = {
  has_subscription: false,
  active: false,
}

export function VpnStatusCard() {
  const { t } = useTranslation()

  /* eslint-disable @tanstack/query/exhaustive-deps */
  const statusQuery = useQuery({
    queryKey: ['vpn-self'],
    queryFn: async () => {
      const res = await getVpnSelf()
      if (res.success) {
        return res.data ?? EMPTY_VPN_STATUS
      }
      throw createServerError(res, t('Failed to load VPN status'))
    },
  })
  /* eslint-enable @tanstack/query/exhaustive-deps */

  if (statusQuery.isLoading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
          <Skeleton className='h-6 w-48' />
          <Skeleton className='mt-2 h-4 w-64' />
        </CardHeader>
        <CardContent className='p-3 sm:p-5'>
          <Skeleton className='h-24 w-full' />
        </CardContent>
      </Card>
    )
  }

  const status = statusQuery.data ?? EMPTY_VPN_STATUS

  return (
    <TitledCard
      title={t('My VPN Subscription')}
      description={t('Your active plan and connection details')}
      icon={<Wifi className='h-4 w-4' />}
      iconTone='info'
      disableHoverEffect
    >
      {statusQuery.isError ? (
        <div className='text-destructive text-sm'>
          {statusQuery.error instanceof Error
            ? statusQuery.error.message
            : t('Failed to load VPN status')}
        </div>
      ) : (
        <VpnStatusBody status={status} />
      )}
    </TitledCard>
  )
}

function VpnStatusBody(props: { status: VpnSelfStatus }) {
  const { t } = useTranslation()
  const status = props.status

  if (!status.has_subscription) {
    return (
      <div className='bg-muted/30 flex items-start gap-3 rounded-lg border p-3 sm:p-4'>
        <ShieldOff className='text-muted-foreground mt-0.5 size-4 shrink-0' />
        <p className='text-muted-foreground text-sm'>
          {t(
            'You do not have an active VPN subscription. Choose a plan below to get started.'
          )}
        </p>
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='min-w-0'>
          <div className='truncate font-medium'>
            {status.plan_title || t('VPN Subscription')}
          </div>
          <div className='text-muted-foreground mt-0.5 text-xs'>
            {t('Data Cap')}: {formatVpnDataCap(status.total_gb || 0, t)}
          </div>
        </div>
        {status.active ? (
          <StatusBadge label={t('Active')} variant='success' copyable={false} />
        ) : (
          <StatusBadge
            label={t('Expired')}
            variant='neutral'
            copyable={false}
          />
        )}
      </div>

      <div className='text-muted-foreground text-xs'>
        {t('Expires')}: {formatVpnTimestamp(status.end_time)}
      </div>

      {status.active && status.share_link && (
        <div className='space-y-3 rounded-lg border p-3 sm:p-4'>
          <div className='flex items-center gap-2'>
            <ShieldCheck className='text-success size-4 shrink-0' />
            <span className='text-sm font-medium'>
              {t('Import this link into your VPN app')}
            </span>
          </div>
          <div className='flex items-center justify-center rounded-lg bg-white p-4'>
            <QRCodeSVG value={status.share_link} size={180} />
          </div>
          <div className='flex items-center gap-2'>
            <code className='bg-muted flex-1 truncate rounded-md px-2 py-1.5 font-mono text-xs'>
              {status.share_link}
            </code>
            <CopyButton
              value={status.share_link}
              variant='outline'
              tooltip={t('Copy link')}
              aria-label={t('Copy link')}
            />
          </div>
        </div>
      )}

      {status.active && !status.share_link && status.share_link_error && (
        <Alert variant='destructive'>
          <AlertDescription>
            {t('Failed to generate your connection link')}:{' '}
            {status.share_link_error}
          </AlertDescription>
        </Alert>
      )}
    </div>
  )
}
