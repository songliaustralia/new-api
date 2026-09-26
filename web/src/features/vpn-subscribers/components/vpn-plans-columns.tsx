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
import type { ColumnDef } from '@tanstack/react-table'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { BadgeCell } from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'

import { formatVpnDataCap } from '../lib'
import type { VpnPlan } from '../types'
import { VpnPlanRowActions } from './vpn-plan-row-actions'

export function useVpnPlansColumns(): ColumnDef<VpnPlan>[] {
  const { t } = useTranslation()

  return useMemo(
    (): ColumnDef<VpnPlan>[] => [
      {
        accessorKey: 'id',
        id: 'id',
        header: t('ID'),
        meta: { mobileHidden: true },
        cell: ({ row }) => <TableId value={row.original.id} />,
        size: 60,
      },
      {
        accessorKey: 'title',
        id: 'title',
        header: t('Plan'),
        meta: { mobileTitle: true },
        cell: ({ row }) => (
          <div className='max-w-full min-w-0'>
            <div className='truncate font-medium'>{row.original.title}</div>
            {row.original.subtitle && (
              <div className='text-muted-foreground truncate text-xs'>
                {row.original.subtitle}
              </div>
            )}
          </div>
        ),
        size: 200,
      },
      {
        accessorKey: 'price_amount',
        id: 'price',
        header: t('Price'),
        cell: ({ row }) => (
          <span className='font-semibold text-emerald-600'>
            {Number(row.original.price_amount || 0).toFixed(2)}{' '}
            {(row.original.currency || 'usd').toUpperCase()}
          </span>
        ),
        size: 120,
      },
      {
        id: 'duration',
        header: t('Validity'),
        cell: ({ row }) => (
          <span className='text-muted-foreground'>
            {t('{{count}} days', { count: row.original.duration_days })}
          </span>
        ),
        size: 100,
      },
      {
        id: 'total_gb',
        header: t('Data Cap'),
        meta: { mobileHidden: true },
        cell: ({ row }) => (
          <span className='text-muted-foreground'>
            {formatVpnDataCap(row.original.total_gb, t)}
          </span>
        ),
        size: 100,
      },
      {
        accessorKey: 'sort_order',
        id: 'sort_order',
        header: t('Priority'),
        meta: { mobileHidden: true },
        cell: ({ row }) => (
          <span className='text-muted-foreground'>
            {row.original.sort_order}
          </span>
        ),
        size: 90,
      },
      {
        accessorKey: 'enabled',
        id: 'enabled',
        header: t('Status'),
        meta: { mobileBadge: true },
        cell: ({ row }) =>
          row.original.enabled ? (
            <StatusBadge
              label={t('Enable')}
              variant='success'
              copyable={false}
              className='-ml-1.5'
            />
          ) : (
            <StatusBadge
              label={t('Disable')}
              variant='neutral'
              copyable={false}
              className='-ml-1.5'
            />
          ),
        size: 80,
      },
      {
        id: 'payment',
        header: t('Payment Channel'),
        meta: { mobileHidden: true },
        cell: ({ row }) =>
          row.original.stripe_price_id ? (
            <BadgeCell>
              <StatusBadge label='Stripe' variant='neutral' copyable={false} />
            </BadgeCell>
          ) : (
            <span className='text-muted-foreground'>-</span>
          ),
        size: 100,
      },
      {
        id: 'actions',
        header: () => t('Actions'),
        cell: ({ row }) => <VpnPlanRowActions row={row} />,
        meta: { pinned: 'right' as const },
      },
    ],
    [t]
  )
}
