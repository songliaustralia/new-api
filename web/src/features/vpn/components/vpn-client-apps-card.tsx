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
import { Download, Laptop, Smartphone } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { TitledCard } from '@/components/ui/titled-card'

interface VpnClientApp {
  id: string
  platformKey: string
  appName: string
  noteKey: string
  url: string
  icon: React.ReactNode
}

// Official / primary distribution links, checked 2026-09-26.
const VPN_CLIENT_APPS: VpnClientApp[] = [
  {
    id: 'ios',
    platformKey: 'iOS (iPhone/iPad)',
    appName: 'Shadowrocket',
    noteKey: 'Paid app on the App Store (one-time purchase)',
    url: 'https://apps.apple.com/us/app/shadowrocket/id932747118',
    icon: <Smartphone className='h-4 w-4 shrink-0' />,
  },
  {
    id: 'android',
    platformKey: 'Android',
    appName: 'NekoBox for Android',
    noteKey: 'Free, open source',
    url: 'https://github.com/MatsuriDayo/NekoBoxForAndroid/releases',
    icon: <Smartphone className='h-4 w-4 shrink-0' />,
  },
  {
    id: 'windows',
    platformKey: 'Windows',
    appName: 'v2rayN',
    noteKey: 'Free, open source',
    url: 'https://github.com/2dust/v2rayN/releases',
    icon: <Laptop className='h-4 w-4 shrink-0' />,
  },
  {
    id: 'macos',
    platformKey: 'macOS',
    appName: 'V2rayU',
    noteKey: 'Free, open source',
    url: 'https://github.com/yanue/V2rayU/releases',
    icon: <Laptop className='h-4 w-4 shrink-0' />,
  },
]

export function VpnClientAppsCard() {
  const { t } = useTranslation()

  return (
    <TitledCard
      title={t('Download a VPN Client')}
      description={t(
        'Install one of these on your device first, then import the link above'
      )}
      icon={<Download className='h-4 w-4' />}
      iconTone='primary'
      disableHoverEffect
    >
      <div className='divide-y'>
        {VPN_CLIENT_APPS.map((app) => (
          <div
            key={app.id}
            className='flex flex-wrap items-center justify-between gap-3 py-3 first:pt-0 last:pb-0'
          >
            <div className='flex min-w-0 items-center gap-3'>
              <div className='bg-muted text-muted-foreground flex size-8 shrink-0 items-center justify-center rounded-lg'>
                {app.icon}
              </div>
              <div className='min-w-0'>
                <div className='flex flex-wrap items-baseline gap-x-2'>
                  <span className='text-sm font-medium'>{app.appName}</span>
                  <span className='text-muted-foreground text-xs'>
                    {t(app.platformKey)}
                  </span>
                </div>
                <p className='text-muted-foreground text-xs'>
                  {t(app.noteKey)}
                </p>
              </div>
            </div>
            <Button
              variant='outline'
              size='sm'
              onClick={() => window.open(app.url, '_blank', 'noopener')}
            >
              {t('Download')}
            </Button>
          </div>
        ))}
      </div>
    </TitledCard>
  )
}
