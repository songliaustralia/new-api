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
import { z } from 'zod'

// The backend re-validates length and provider on every submit; this schema
// only exists to give the customer an immediate, friendly hint before the
// request round-trip.
export const byokKeyFormSchema = z.object({
  key: z
    .string()
    .trim()
    .min(20, { message: 'This key looks too short' })
    .max(512, { message: 'This key looks too long' }),
})

export type ByokKeyFormValues = z.infer<typeof byokKeyFormSchema>
