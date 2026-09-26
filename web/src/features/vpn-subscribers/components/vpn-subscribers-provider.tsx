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
import React, { useState } from 'react'

import useDialogState from '@/hooks/use-dialog'

import type {
  VpnPlan,
  VpnPlanDialogType,
  VpnSubscriber,
  VpnSubscriberDialogType,
} from '../types'

type VpnSubscribersContextType = {
  planDialog: VpnPlanDialogType | null
  setPlanDialog: (value: VpnPlanDialogType | null) => void
  currentPlan: VpnPlan | null
  setCurrentPlan: React.Dispatch<React.SetStateAction<VpnPlan | null>>
  plansRefreshTrigger: number
  triggerPlansRefresh: () => void

  subscriberDialog: VpnSubscriberDialogType | null
  setSubscriberDialog: (value: VpnSubscriberDialogType | null) => void
  currentSubscriber: VpnSubscriber | null
  setCurrentSubscriber: React.Dispatch<
    React.SetStateAction<VpnSubscriber | null>
  >
  subscribersRefreshTrigger: number
  triggerSubscribersRefresh: () => void
}

const VpnSubscribersContext =
  React.createContext<VpnSubscribersContextType | null>(null)

export function VpnSubscribersProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const [planDialog, setPlanDialog] = useDialogState<VpnPlanDialogType>(null)
  const [currentPlan, setCurrentPlan] = useState<VpnPlan | null>(null)
  const [plansRefreshTrigger, setPlansRefreshTrigger] = useState(0)

  const [subscriberDialog, setSubscriberDialog] =
    useDialogState<VpnSubscriberDialogType>(null)
  const [currentSubscriber, setCurrentSubscriber] =
    useState<VpnSubscriber | null>(null)
  const [subscribersRefreshTrigger, setSubscribersRefreshTrigger] = useState(0)

  const triggerPlansRefresh = () => setPlansRefreshTrigger((prev) => prev + 1)
  const triggerSubscribersRefresh = () =>
    setSubscribersRefreshTrigger((prev) => prev + 1)

  return (
    <VpnSubscribersContext
      value={{
        planDialog,
        setPlanDialog,
        currentPlan,
        setCurrentPlan,
        plansRefreshTrigger,
        triggerPlansRefresh,
        subscriberDialog,
        setSubscriberDialog,
        currentSubscriber,
        setCurrentSubscriber,
        subscribersRefreshTrigger,
        triggerSubscribersRefresh,
      }}
    >
      {children}
    </VpnSubscribersContext>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export const useVpnSubscribers = () => {
  const ctx = React.useContext(VpnSubscribersContext)
  if (!ctx) {
    throw new Error(
      'useVpnSubscribers has to be used within <VpnSubscribersProvider>'
    )
  }
  return ctx
}
