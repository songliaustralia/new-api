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
import { DeletePlanDialog } from './dialogs/delete-plan-dialog'
import { TogglePlanStatusDialog } from './dialogs/toggle-plan-status-dialog'
import { VpnPlansMutateDrawer } from './vpn-plans-mutate-drawer'
import { useVpnSubscribers } from './vpn-subscribers-provider'

export function VpnPlansDialogs() {
  const { planDialog, setPlanDialog, currentPlan } = useVpnSubscribers()
  const isUpdate = planDialog === 'update'

  return (
    <>
      <VpnPlansMutateDrawer
        open={planDialog === 'create' || isUpdate}
        onOpenChange={(isOpen) => !isOpen && setPlanDialog(null)}
        currentRow={isUpdate ? currentPlan || undefined : undefined}
      />
      <TogglePlanStatusDialog />
      <DeletePlanDialog />
    </>
  )
}
