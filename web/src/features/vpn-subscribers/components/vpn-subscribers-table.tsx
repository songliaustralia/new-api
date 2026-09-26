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
import { Ban, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  DataTableRowActionMenu,
  StaticDataTable,
} from '@/components/data-table'
import { TableId } from '@/components/table-id'
import {
  DropdownMenuItem,
  DropdownMenuShortcut,
} from '@/components/ui/dropdown-menu'
import { requireServerSuccess } from '@/lib/server-error-message'

import { getVpnSubscribers } from '../api'
import { formatVpnDataCap, formatVpnTimestamp } from '../lib'
import type { VpnSubscriber } from '../types'
import { VpnSubscriberStatusBadge } from './vpn-subscriber-status-badge'
import { useVpnSubscribers } from './vpn-subscribers-provider'

export function VpnSubscribersTable() {
  const { t } = useTranslation()
  const {
    subscribersRefreshTrigger,
    setSubscriberDialog,
    setCurrentSubscriber,
  } = useVpnSubscribers()

  const { data, isLoading } = useQuery({
    queryKey: ['vpn-admin-subscribers', subscribersRefreshTrigger],
    queryFn: async () => {
      const result = requireServerSuccess(await getVpnSubscribers())
      return result.data || []
    },
    placeholderData: (prev) => prev,
  })

  const subscribers = data || []

  const handleGrant = (subscriber: VpnSubscriber) => {
    setCurrentSubscriber(subscriber)
    setSubscriberDialog('grant')
  }

  const handleRevoke = (subscriber: VpnSubscriber) => {
    setCurrentSubscriber(subscriber)
    setSubscriberDialog('revoke')
  }

  return (
    <StaticDataTable<VpnSubscriber>
      data={isLoading ? [] : subscribers}
      getRowKey={(subscriber) => subscriber.id}
      emptyClassName={isLoading ? 'py-8' : 'text-muted-foreground py-8'}
      emptyContent={isLoading ? t('Loading...') : t('No subscribers yet')}
      columns={[
        {
          id: 'id',
          header: t('ID'),
          cell: (subscriber) => <TableId value={subscriber.id} />,
        },
        {
          id: 'user',
          header: t('User'),
          cell: (subscriber) => (
            <div>
              <div className='font-medium'>{subscriber.username || '-'}</div>
              <div className='text-muted-foreground text-xs'>
                {t('Plan')}: {subscriber.plan_title || `#${subscriber.plan_id}`}
              </div>
            </div>
          ),
        },
        {
          id: 'status',
          header: t('Status'),
          cell: (subscriber) => (
            <VpnSubscriberStatusBadge subscriber={subscriber} />
          ),
        },
        {
          id: 'total_gb',
          header: t('Data Cap'),
          cell: (subscriber) => formatVpnDataCap(subscriber.total_gb, t),
        },
        {
          id: 'validity',
          header: t('Validity'),
          cell: (subscriber) => (
            <div className='text-sm'>
              <div>
                {t('Start')}: {formatVpnTimestamp(subscriber.start_time)}
              </div>
              <div>
                {t('End')}: {formatVpnTimestamp(subscriber.end_time)}
              </div>
            </div>
          ),
        },
        {
          id: 'actions',
          header: t('Actions'),
          className: 'text-right',
          cellClassName: 'text-right',
          cell: (subscriber) => (
            <DataTableRowActionMenu ariaLabel={t('Actions')}>
              <DropdownMenuItem onClick={() => handleGrant(subscriber)}>
                {t('Grant/Extend')}
                <DropdownMenuShortcut>
                  <RefreshCw size={16} />
                </DropdownMenuShortcut>
              </DropdownMenuItem>
              <DropdownMenuItem
                variant='destructive'
                disabled={subscriber.status === 'revoked'}
                onClick={() => handleRevoke(subscriber)}
              >
                {t('Revoke')}
                <DropdownMenuShortcut>
                  <Ban size={16} />
                </DropdownMenuShortcut>
              </DropdownMenuItem>
            </DataTableRowActionMenu>
          ),
        },
      ]}
    />
  )
}
