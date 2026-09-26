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
import { Check, Package } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import { handleServerError } from '@/lib/handle-server-error'
import { requireServerSuccess } from '@/lib/server-error-message'

import { getVpnPlans, payVpnStripe } from '../api'
import { formatVpnDataCap, formatVpnDuration } from '../lib'
import type { VpnPlan } from '../types'

export function VpnPlansGrid() {
  const { t } = useTranslation()
  const [payingPlanId, setPayingPlanId] = useState<number | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['vpn-plans'],
    queryFn: async () => {
      const result = requireServerSuccess(await getVpnPlans())
      return result.data || []
    },
  })

  const plans = data || []

  const handleSubscribe = async (plan: VpnPlan) => {
    setPayingPlanId(plan.id)
    try {
      const res = await payVpnStripe({ plan_id: plan.id })
      if (res.success && res.data?.pay_link) {
        window.open(res.data.pay_link, '_blank')
        toast.success(t('Payment page opened'))
      } else {
        handleServerError(res, t('Payment request failed'))
      }
    } catch (error) {
      handleServerError(error, t('Payment request failed'))
    } finally {
      setPayingPlanId(null)
    }
  }

  function renderContent() {
    if (isLoading) {
      return (
        <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3'>
          {['a', 'b', 'c'].map((key) => (
            <Skeleton key={key} className='h-48 w-full' />
          ))}
        </div>
      )
    }

    if (plans.length === 0) {
      return (
        <p className='text-muted-foreground py-4 text-center text-sm'>
          {t('No plans available')}
        </p>
      )
    }

    return (
      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3'>
        {plans.map((plan) => (
          <Card key={plan.id} data-card-hover='false'>
            <CardContent className='flex h-full flex-col p-3.5 sm:p-4'>
              <div className='min-w-0'>
                <h4 className='truncate font-semibold'>{plan.title}</h4>
                {plan.subtitle && (
                  <p className='text-muted-foreground truncate text-xs'>
                    {plan.subtitle}
                  </p>
                )}
              </div>

              <div className='py-2'>
                <span className='text-primary text-2xl font-bold'>
                  ${Number(plan.price_amount || 0).toFixed(2)}
                </span>
              </div>

              <div className='flex-1 space-y-1.5 pb-3'>
                <div className='text-muted-foreground flex items-center gap-2 text-xs'>
                  <Check className='text-primary h-3 w-3 shrink-0' />
                  <span>
                    {t('Validity Period')}:{' '}
                    {formatVpnDuration(plan.duration_days, t)}
                  </span>
                </div>
                <div className='text-muted-foreground flex items-center gap-2 text-xs'>
                  <Check className='text-primary h-3 w-3 shrink-0' />
                  <span>
                    {t('Data Cap')}: {formatVpnDataCap(plan.total_gb, t)}
                  </span>
                </div>
              </div>

              <Separator className='mb-3' />

              <Button
                variant='outline'
                className='w-full'
                onClick={() => handleSubscribe(plan)}
                disabled={payingPlanId === plan.id}
              >
                {payingPlanId === plan.id
                  ? t('Redirecting...')
                  : t('Subscribe')}
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  return (
    <TitledCard
      title={t('Plans')}
      description={t('Pick a plan and pay securely via Stripe')}
      icon={<Package className='h-4 w-4' />}
      iconTone='primary'
      disableHoverEffect
    >
      {renderContent()}
    </TitledCard>
  )
}
