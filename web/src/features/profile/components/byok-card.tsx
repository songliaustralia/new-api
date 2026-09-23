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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { KeyRound, Loader2, Lock, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { PasswordInput } from '@/components/password-input'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from '@/components/ui/form'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import { handleServerError } from '@/lib/handle-server-error'
import { createServerError } from '@/lib/server-error-message'

import { deleteByokKey, getByokStatus, setByokKey } from '../api'
import { byokKeyFormSchema, type ByokKeyFormValues } from '../lib/byok-form'
import type { ByokProviderId, ByokProviderStatus, ByokStatus } from '../types'

const EMPTY_BYOK_STATUS: ByokStatus = { eligible: false, providers: {} }

// ============================================================================
// BYOK (Bring Your Own Key) Card Component
// ============================================================================

const BYOK_STATUS_QUERY_KEY = ['byok-status']

const BYOK_PROVIDERS: Array<{ id: ByokProviderId; label: string }> = [
  { id: 'openai', label: 'OpenAI' },
  { id: 'anthropic', label: 'Anthropic (Claude)' },
]

export function ByokCard() {
  const { t } = useTranslation()

  /* eslint-disable @tanstack/query/exhaustive-deps */
  const statusQuery = useQuery({
    queryKey: BYOK_STATUS_QUERY_KEY,
    queryFn: async () => {
      const res = await getByokStatus()
      if (res.success) {
        return res.data ?? EMPTY_BYOK_STATUS
      }
      throw createServerError(res, t('Failed to load key status'))
    },
  })
  /* eslint-enable @tanstack/query/exhaustive-deps */

  if (statusQuery.isLoading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
          <Skeleton className='h-6 w-40' />
          <Skeleton className='mt-2 h-4 w-56' />
        </CardHeader>
        <CardContent className='space-y-3 p-3 sm:p-5'>
          <Skeleton className='h-24 w-full' />
          <Skeleton className='h-24 w-full' />
        </CardContent>
      </Card>
    )
  }

  const status = statusQuery.data ?? EMPTY_BYOK_STATUS

  function renderBody() {
    if (statusQuery.isError) {
      const message =
        statusQuery.error instanceof Error
          ? statusQuery.error.message
          : t('Failed to load key status')
      return <div className='text-destructive text-sm'>{message}</div>
    }

    if (!status.eligible) {
      return (
        <div className='bg-muted/30 flex items-start gap-3 rounded-lg border p-3 sm:p-4'>
          <Lock className='text-muted-foreground mt-0.5 size-4 shrink-0' />
          <p className='text-muted-foreground text-sm'>
            {t(
              'Bring Your Own Key is included with any paid subscription plan. Subscribe to a plan to unlock it.'
            )}
          </p>
        </div>
      )
    }

    return (
      <div className='space-y-3'>
        {BYOK_PROVIDERS.map((provider) => (
          <ByokProviderRow
            key={provider.id}
            providerId={provider.id}
            label={provider.label}
            status={status.providers[provider.id]}
          />
        ))}
      </div>
    )
  }

  return (
    <TitledCard
      title={t('Bring Your Own Key')}
      description={t(
        'Use your own OpenAI or Anthropic API key. Requests you make will be billed to your own key, not your account balance.'
      )}
      icon={<KeyRound className='h-4 w-4' />}
      iconTone='info'
      disableHoverEffect
    >
      {renderBody()}
    </TitledCard>
  )
}

// ============================================================================
// Single Provider Row
// ============================================================================

interface ByokProviderRowProps {
  providerId: ByokProviderId
  label: string
  status?: ByokProviderStatus
}

function ByokProviderRow({ providerId, label, status }: ByokProviderRowProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)

  const form = useForm<ByokKeyFormValues>({
    resolver: zodResolver(byokKeyFormSchema),
    defaultValues: { key: '' },
  })

  const setKey = useMutation({
    mutationFn: async (values: ByokKeyFormValues) => {
      const res = await setByokKey({ provider: providerId, key: values.key })
      if (!res.success) {
        throw createServerError(res, t('Failed to save your key'))
      }
    },
    onSuccess: () => {
      toast.success(t('Your key has been saved'))
      form.reset({ key: '' })
      queryClient.invalidateQueries({ queryKey: BYOK_STATUS_QUERY_KEY })
    },
    onError: (error) => handleServerError(error, t('Failed to save your key')),
  })

  const removeKey = useMutation({
    mutationFn: async () => {
      const res = await deleteByokKey(providerId)
      if (!res.success) {
        throw createServerError(res, t('Failed to remove your key'))
      }
    },
    onSuccess: () => {
      toast.success(t('Your key has been removed'))
      setConfirmOpen(false)
      queryClient.invalidateQueries({ queryKey: BYOK_STATUS_QUERY_KEY })
    },
    onError: (error) => {
      handleServerError(error, t('Failed to remove your key'))
      setConfirmOpen(false)
    },
  })

  const configured = status?.configured === true

  return (
    <div className='rounded-lg border p-3 sm:p-4'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='flex flex-wrap items-center gap-2'>
          <span className='text-sm font-medium'>{label}</span>
          {configured ? (
            <Badge variant='secondary'>
              {status?.key_masked
                ? `${t('Configured')} · ${status.key_masked}`
                : t('Configured')}
            </Badge>
          ) : (
            <Badge variant='outline'>{t('Not configured')}</Badge>
          )}
        </div>
        {configured && (
          <Button
            type='button'
            variant='ghost'
            size='sm'
            className='text-destructive hover:text-destructive'
            onClick={() => setConfirmOpen(true)}
            disabled={removeKey.isPending}
          >
            <Trash2 className='size-4' />
            {t('Remove')}
          </Button>
        )}
      </div>

      <Form {...form}>
        <form
          className='mt-3 flex flex-col gap-2 sm:flex-row sm:items-start'
          onSubmit={form.handleSubmit((values) => setKey.mutate(values))}
        >
          <FormField
            control={form.control}
            name='key'
            render={({ field }) => (
              <FormItem className='flex-1'>
                <FormControl>
                  <PasswordInput
                    placeholder={
                      configured
                        ? t('Enter a new key to replace the current one')
                        : t('Paste your API key')
                    }
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button
            type='submit'
            disabled={setKey.isPending}
            className='shrink-0'
          >
            {setKey.isPending && <Loader2 className='size-4 animate-spin' />}
            {configured ? t('Replace') : t('Save')}
          </Button>
        </form>
      </Form>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t('Remove this key?')}
        desc={t(
          'Requests that use this key will stop working immediately. You can add a new key at any time.'
        )}
        destructive
        isLoading={removeKey.isPending}
        handleConfirm={() => removeKey.mutate()}
      />
    </div>
  )
}
