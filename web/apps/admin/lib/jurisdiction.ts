// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: browsing/creating jurisdiction units (M4.1, D-JurisdictionUnits) — M10.7 repoints
// this from go-oikumenea's Tenant/Religion services (lib/oikumenea.ts, deleted this milestone) to
// openfaithmap-api's own core.conjure.yml surface (lib/core.ts). No "organization" concept survives
// the port (internal/directory is single-tenant, D-CorePortScope) — searchJurisdictionUnits is now a
// direct code/name search over every unit, not an org-then-list-units dance, genuinely simpler than
// what it replaces, not just repointed.
import "server-only";

import * as core from "./core";

export type JurisdictionUnit = {
  id: string;
  code: string | null;
  name: string;
};

function toJurisdictionUnit(u: { id: string; code?: string | null; name: string }): JurisdictionUnit {
  return { id: u.id, code: u.code ?? null, name: u.name };
}

/**
 * Free-text search over every unit's code/name (D-JurisdictionUnits deliberately doesn't restrict
 * this to jurisdiction-tagged org kinds or a fixed depth — jurisdiction is operator-judgment-driven,
 * variable depth, and not every denomination's structure fits one vocabulary). The operator narrows
 * by typing a name; this is not meant to browse the whole tree.
 */
export async function searchJurisdictionUnits(query: string): Promise<JurisdictionUnit[]> {
  const trimmed = query.trim();
  if (!trimmed) return [];
  const units = await core.listUnits(trimmed, 20);
  return units.map(toJurisdictionUnit);
}

/** Ancestor breadcrumb for unitId, nearest first, so the operator can see what they're attaching to. */
export async function jurisdictionUnitAncestors(unitId: string): Promise<JurisdictionUnit[]> {
  const refs = await core.unitAncestors(unitId);
  return refs.map(toJurisdictionUnit);
}

/**
 * Creates a new jurisdiction unit as a child of parentUnitId (root, or another jurisdiction unit,
 * for nesting) — a plain, uncoupled write, unlike congregation approval's multi-step sequence. Gated
 * server-side: the caller must hold religionorg.manage over parentUnitId.
 */
export async function createJurisdictionUnit(
  parentUnitId: string,
  code: string,
  name: string,
): Promise<JurisdictionUnit> {
  const profile = await core.createChildOrg({ parentUnitId, code, name });
  return { id: profile.unitId, code, name };
}
