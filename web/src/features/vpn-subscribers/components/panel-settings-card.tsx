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
import { CheckCircle2, Loader2, Server, XCircle } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { PasswordInput } from '@/components/password-input'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { TitledCard } from '@/components/ui/titled-card'
import {
  getOptionValue,
  useSystemOptions,
} from '@/features/system-settings/hooks/use-system-options'
import { useUpdateOption } from '@/features/system-settings/hooks/use-update-option'
import { handleServerError } from '@/lib/handle-server-error'

import { testVpnPanelConnection } from '../api'
import type { VpnPanelTestResult } from '../types'

const PANEL_OPTION_DEFAULTS = {
  VpnFeatureEnabled: false,
  VpnPanelBaseUrl: '',
  VpnPanelApiToken: '',
  VpnPanelInboundId: '',
  VpnServerHost: '',
  VpnServerPort: '443',
  VpnNetwork: 'tcp',
  VpnRealityPublicKey: '',
  VpnRealitySni: '',
  VpnRealityShortId: '',
  VpnRealitySpiderX: '',
  VpnRealityFingerprint: 'chrome',
  VpnRealityFlow: '',
}

type PanelOptions = typeof PANEL_OPTION_DEFAULTS

const panelSettingsSchema = z.object({
  VpnFeatureEnabled: z.boolean(),
  VpnPanelBaseUrl: z.string(),
  VpnPanelApiToken: z.string(),
  VpnPanelInboundId: z.string(),
  VpnServerHost: z.string(),
  VpnServerPort: z.string(),
  VpnNetwork: z.string(),
  VpnRealityPublicKey: z.string(),
  VpnRealitySni: z.string(),
  VpnRealityShortId: z.string(),
  VpnRealitySpiderX: z.string(),
  VpnRealityFingerprint: z.string(),
  VpnRealityFlow: z.string(),
})

type PanelSettingsFormValues = z.infer<typeof panelSettingsSchema>

export function PanelSettingsCard() {
  const { t } = useTranslation()
  const { data, isLoading } = useSystemOptions()
  const updateOption = useUpdateOption()
  const [isSaving, setIsSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<
    | { success: true; data: VpnPanelTestResult }
    | { success: false; message: string }
    | null
  >(null)

  const options = useMemo(
    () => getOptionValue(data?.data, PANEL_OPTION_DEFAULTS),
    [data?.data]
  )
  const baselineRef = useRef<PanelOptions>(options)

  const form = useForm<PanelSettingsFormValues>({
    resolver: zodResolver(panelSettingsSchema),
    defaultValues: options,
  })

  useEffect(() => {
    baselineRef.current = options
    form.reset(options)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [options])

  const onSubmit = async (values: PanelSettingsFormValues) => {
    const keys = Object.keys(values) as Array<keyof PanelOptions>
    const changed = keys.filter(
      (key) => values[key] !== baselineRef.current[key]
    )

    if (changed.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    setIsSaving(true)
    try {
      for (const key of changed) {
        await updateOption.mutateAsync({ key, value: values[key] })
      }
      baselineRef.current = values
    } finally {
      setIsSaving(false)
    }
  }

  const handleTestConnection = async () => {
    setTesting(true)
    setTestResult(null)
    try {
      const res = await testVpnPanelConnection()
      if (res.success && res.data) {
        setTestResult({ success: true, data: res.data })
      } else {
        setTestResult({
          success: false,
          message: res.message || t('Connection test failed'),
        })
      }
    } catch (error) {
      handleServerError(error, t('Connection test failed'))
      setTestResult({
        success: false,
        message: t('Connection test failed'),
      })
    } finally {
      setTesting(false)
    }
  }

  if (isLoading) {
    return (
      <TitledCard
        title={t('Panel Settings')}
        icon={<Server className='h-4 w-4' />}
        disableHoverEffect
      >
        <div className='space-y-3'>
          <Skeleton className='h-9 w-full' />
          <Skeleton className='h-9 w-full' />
          <Skeleton className='h-9 w-full' />
        </div>
      </TitledCard>
    )
  }

  return (
    <TitledCard
      title={t('Panel Settings')}
      description={t('Connect this feature to your 3x-ui panel')}
      icon={<Server className='h-4 w-4' />}
      iconTone='primary'
      disableHoverEffect
      contentClassName='space-y-5'
    >
      <Form {...form}>
        <form
          id='vpn-panel-settings-form'
          onSubmit={form.handleSubmit(onSubmit)}
          className='space-y-4'
        >
          <FormField
            control={form.control}
            name='VpnFeatureEnabled'
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-md border px-3 py-2'>
                <FormLabel className='!mt-0'>
                  {t('VPN feature enabled')}
                </FormLabel>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <div className='grid grid-cols-1 gap-3 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name='VpnPanelBaseUrl'
              render={({ field }) => (
                <FormItem className='sm:col-span-2'>
                  <FormLabel>{t('Panel Base URL')}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      placeholder='https://108.61.161.5:2053/4iV2FcjbFgILregyHc/'
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnPanelApiToken'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Panel API Token')}</FormLabel>
                  <FormControl>
                    <PasswordInput {...field} autoComplete='off' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnPanelInboundId'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Panel Inbound ID')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder='1' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnServerHost'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Server Host')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder='108.61.161.5' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnServerPort'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Server Port')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder='443' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnNetwork'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Network')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder='tcp' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnRealityPublicKey'
              render={({ field }) => (
                <FormItem className='sm:col-span-2'>
                  <FormLabel>{t('Reality Public Key')}</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnRealitySni'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Reality SNI')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder='www.microsoft.com' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnRealityShortId'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Reality Short ID')}</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnRealitySpiderX'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Reality SpiderX')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder='/' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnRealityFingerprint'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Reality Fingerprint')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder='chrome' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='VpnRealityFlow'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Reality Flow')}</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder='xtls-rprx-vision' />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <div className='flex flex-wrap items-center gap-2 pt-1'>
            <Button
              type='submit'
              form='vpn-panel-settings-form'
              disabled={isSaving}
            >
              {isSaving && <Loader2 className='size-4 animate-spin' />}
              {isSaving ? t('Saving...') : t('Save changes')}
            </Button>
            <Button
              type='button'
              variant='outline'
              onClick={handleTestConnection}
              disabled={testing}
            >
              {testing && <Loader2 className='size-4 animate-spin' />}
              {testing ? t('Testing...') : t('Test Connection')}
            </Button>
          </div>

          {testResult && (
            <Alert variant={testResult.success ? 'default' : 'destructive'}>
              {testResult.success ? (
                <CheckCircle2 className='h-4 w-4' />
              ) : (
                <XCircle className='h-4 w-4' />
              )}
              <AlertDescription>
                {testResult.success
                  ? t(
                      'Connected. Inbound "{{remark}}" on port {{port}} ({{protocol}}).',
                      {
                        remark: testResult.data.remark,
                        port: testResult.data.port,
                        protocol: testResult.data.protocol,
                      }
                    )
                  : testResult.message}
              </AlertDescription>
            </Alert>
          )}
        </form>
      </Form>
    </TitledCard>
  )
}
