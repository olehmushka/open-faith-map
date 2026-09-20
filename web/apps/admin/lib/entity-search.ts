// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use server";

// Server Actions backing EntityPicker's onSearch prop (M17) — one shared module rather than a
// per-page inline wrapper, since the same three searches (unit/person/taxon) are reused across
// most of web/apps/admin. A plain function passed from a Server Component into a Client Component
// (EntityPicker) must be a Server Action, which is what the "use server" file directive gives
// every export below — lib/core.ts's own search wrappers stay server-only, unreachable directly
// from client code.
//
// searchPersonsForPicker calls core.searchPersons, which is CoreSuperAdminService-backed and
// therefore instance-admin-gated (RequireInstanceAdmin, cmd/openfaithmap-api/register_core.go) —
// only wire it into pages that already live under (super-admin). A non-instance-admin caller
// would get Forbidden, which is why vouching's claimant/guarantor-person fields (open to any
// logged-in operator, no local gate) were deliberately left as plain <Input> rather than wired to
// this — see docs/milestones-2026-09-20-now.md's M17 detail section.
import { listUnits, listTaxa, searchPersons } from "./core";
import type { EntityPickerOption } from "@/components/entity-picker";

export async function searchUnitsForPicker(query: string): Promise<EntityPickerOption[]> {
  const units = await listUnits(query, 20);
  return units.map((u) => ({ id: u.id, label: u.name, sublabel: u.code ?? undefined }));
}

export async function searchPersonsForPicker(query: string): Promise<EntityPickerOption[]> {
  const persons = await searchPersons(query, 50);
  return persons.map((p) => ({ id: p.id, label: p.displayName, sublabel: p.code ?? undefined }));
}

export async function searchTaxaForPicker(query: string): Promise<EntityPickerOption[]> {
  const taxa = await listTaxa(query, 50);
  return taxa.map((t) => ({ id: t.id, label: t.name, sublabel: t.code }));
}
