// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: browsing/creating jurisdiction units (M4.1, D-JurisdictionUnits) goes straight to
// go-oikumenea's Tenant/Religion services via lib/oikumenea.ts's client — no new openfaithmap-api
// endpoint. Unlike the discovery module's cache, this is a low-traffic, operator-only admin control,
// so a second local mirror of go-oikumenea's own graph would just be a driftable source of truth for
// no real benefit (D-Facade).
import "server-only";

import { oikumenea } from "./oikumenea";

function requireRootUnitId(): string {
  const raw = process.env.REGISTRATION_ROOT_UNIT_ID?.trim();
  if (!raw) {
    throw new Error("REGISTRATION_ROOT_UNIT_ID is not set (see web/apps/admin/.env.example).");
  }
  return raw;
}

export type JurisdictionUnit = {
  id: string;
  code: string | null;
  name: string;
};

function displayName(name: Record<string, string>): string {
  return name["en"] ?? Object.values(name)[0] ?? "(unnamed)";
}

/**
 * Free-text search over every unit in the root organization (D-JurisdictionUnits deliberately
 * doesn't restrict this to jurisdiction-tagged org kinds or a fixed depth — jurisdiction is
 * operator-judgment-driven, variable depth, and not every denomination's structure fits one
 * vocabulary). The operator narrows by typing a name; this is not meant to browse the whole tree.
 */
export async function searchJurisdictionUnits(query: string): Promise<JurisdictionUnit[]> {
  const trimmed = query.trim();
  if (!trimmed) return [];
  const client = await oikumenea();
  const rootUnitId = requireRootUnitId();
  const root = await client.tenant.getUnit(rootUnitId);
  const page = await client.tenant.listUnits(
    root.orgId,
    trimmed,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    null,
    "canonical",
    null,
    null,
    20,
    null,
  );
  return page.units.map((u) => ({ id: u.id, code: u.code ?? null, name: displayName(u.name) }));
}

/** Ancestor breadcrumb for unitId, nearest first, so the operator can see what they're attaching to. */
export async function jurisdictionUnitAncestors(unitId: string): Promise<JurisdictionUnit[]> {
  const client = await oikumenea();
  const list = await client.tenant.unitAncestors(unitId, "canonical");
  return list.units.map((u) => ({ id: u.id, code: u.code ?? null, name: displayName(u.name) }));
}

/**
 * Creates a new jurisdiction unit as a child of parentUnitId (root, or another jurisdiction unit,
 * for nesting) — a plain, uncoupled go-oikumenea write, unlike congregation approval's multi-step
 * sequence. Uses the caller's own forwarded token; go-oikumenea's PDP decides for real.
 */
export async function createJurisdictionUnit(
  parentUnitId: string,
  code: string,
  name: string,
): Promise<JurisdictionUnit> {
  const client = await oikumenea();
  const profile = await client.religion.createChildOrg(parentUnitId, { code, name });
  return { id: profile.unitId, code, name };
}
