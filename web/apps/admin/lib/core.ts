// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: openfaithmap-api's core module via the generated TypeScript SDK (./openfaithmap,
// M10.7), same shape as lib/content.ts. Replaces lib/oikumenea.ts (deleted this milestone) — every
// call this admin app used to make against a sibling go-oikumenea instance now goes through
// openfaithmap-api's own Conjure surface instead, forwarding the session's Google ID token the same
// way lib/oikumenea.ts did. The "general" section below needs no extra role beyond a valid session;
// the "super-admin" section is gated server-side by CoreSuperAdminService's whole-route-group
// RequireInstanceAdmin check (internal/authz/transport) — a 403 surfaces as CoreApiError here, same
// as any other typed error.
import "server-only";

import { isConjureError } from "conjure-client";

import { auth } from "@/auth";

import { createOpenFaithMapClient } from "./openfaithmap";
import type {
  IAccountStatus,
  IAuditLogEntry,
  IAuditLogPage,
  ICountry,
  ICreateChildOrgRequest,
  IInstanceAdminGrant,
  IMembership,
  IOrgKind,
  IOrgProfile,
  IPerson,
  IRole,
  IRoleAssignment,
  ITaxon,
  IUnit,
  IUnitRef,
  IWhoami,
} from "./openfaithmap/generated/core";

export type Whoami = IWhoami;
export type Unit = IUnit;
export type UnitRef = IUnitRef;
export type Taxon = ITaxon;
export type OrgKind = IOrgKind;
export type OrgProfile = IOrgProfile;
export type Country = ICountry;
export type Membership = IMembership;
export type Person = IPerson;
export type Role = IRole;
export type RoleAssignment = IRoleAssignment;
export type InstanceAdminGrant = IInstanceAdminGrant;
export type AccountStatus = IAccountStatus;
export type CreateChildOrgInput = ICreateChildOrgRequest;
export type AuditLogEntry = IAuditLogEntry;
export type AuditLogPage = IAuditLogPage;

export class CoreApiError extends Error {
  constructor(
    public status: number,
    public errorName: string,
    public parameters: Record<string, unknown>,
  ) {
    super(`${errorName} (${status})`);
  }
}

function requireBaseUrl(): string {
  const raw = process.env.OPENFAITHMAP_API_BASE_URL?.trim();
  if (!raw) {
    throw new Error("OPENFAITHMAP_API_BASE_URL is not set.");
  }
  return raw.replace(/\/+$/, "");
}

async function client() {
  const session = await auth();
  return createOpenFaithMapClient({
    baseUrl: requireBaseUrl(),
    token: session?.idToken,
  });
}

/** Translates a ConjureError (the SDK's transport-level error) into the errorName/parameters shape callers already handle. */
async function unwrap<T>(promise: Promise<T>): Promise<T> {
  try {
    return await promise;
  } catch (e) {
    if (isConjureError(e) && e.body && typeof e.body === "object") {
      const body = e.body as { errorName?: string; parameters?: Record<string, unknown> };
      throw new CoreApiError(e.status ?? 0, body.errorName ?? "Unknown", body.parameters ?? {});
    }
    throw e;
  }
}

// ---- general (session-gated only, except createChildOrg) ----

export async function whoami(): Promise<Whoami> {
  return unwrap((await client()).core.whoami());
}

export async function getUnit(unitId: string): Promise<Unit> {
  return unwrap((await client()).core.getUnit(unitId));
}

export async function listUnits(query: string, limit = 20): Promise<Unit[]> {
  const page = await unwrap((await client()).core.listUnits(query, limit));
  return page.units;
}

export async function unitAncestors(unitId: string): Promise<UnitRef[]> {
  const page = await unwrap((await client()).core.unitAncestors(unitId));
  return page.units;
}

export async function listTaxa(query?: string, limit = 500): Promise<Taxon[]> {
  const page = await unwrap((await client()).core.listTaxa(query, limit));
  return page.taxa;
}

export async function getTaxon(taxonId: string): Promise<Taxon> {
  return unwrap((await client()).core.getTaxon(taxonId));
}

export async function listOrgKinds(): Promise<OrgKind[]> {
  const page = await unwrap((await client()).core.listOrgKinds());
  return page.orgKinds;
}

export async function getOrgProfile(unitId: string): Promise<OrgProfile> {
  return unwrap((await client()).core.getOrgProfile(unitId));
}

/** Gated server-side — the caller must hold religionorg.manage over parentUnitId. */
export async function createChildOrg(input: CreateChildOrgInput): Promise<OrgProfile> {
  return unwrap((await client()).core.createChildOrg(input));
}

export async function listCountries(): Promise<Country[]> {
  const page = await unwrap((await client()).core.listCountries());
  return page.countries;
}

export async function listMembershipsByUnit(unitId: string): Promise<Membership[]> {
  const page = await unwrap((await client()).core.listMembershipsByUnit(unitId));
  return page.memberships;
}

export async function getPerson(personId: string): Promise<Person> {
  return unwrap((await client()).core.getPerson(personId));
}

/** Batched read — replaces the pre-cutover my-congregation page's per-member getPerson loop. */
export async function getPersons(personIds: string[]): Promise<Person[]> {
  if (personIds.length === 0) return [];
  const page = await unwrap((await client()).core.getPersons({ personIds }));
  return page.persons;
}

// ---- super-admin (gated server-side by RequireInstanceAdmin) ----

export async function searchPersons(query?: string, limit = 50): Promise<Person[]> {
  const page = await unwrap((await client()).coreSuperAdmin.searchPersons(query, limit));
  return page.persons;
}

export async function listRoles(): Promise<Role[]> {
  const page = await unwrap((await client()).coreSuperAdmin.listRoles());
  return page.roles;
}

export async function listRoleAssignmentsByUnit(unitId: string): Promise<RoleAssignment[]> {
  const page = await unwrap((await client()).coreSuperAdmin.listRoleAssignmentsByUnit(unitId));
  return page.assignments;
}

export async function grantUnitRole(personId: string, roleId: string, unitId: string): Promise<void> {
  return unwrap((await client()).coreSuperAdmin.grantUnitRole({ personId, roleId, unitId }));
}

export async function revokeRoleAssignment(assignmentId: string): Promise<void> {
  return unwrap((await client()).coreSuperAdmin.revokeRoleAssignment(assignmentId));
}

export async function listInstanceAdmins(): Promise<InstanceAdminGrant[]> {
  const page = await unwrap((await client()).coreSuperAdmin.listInstanceAdmins());
  return page.admins;
}

export async function grantInstanceAdmin(personId: string): Promise<InstanceAdminGrant> {
  return unwrap((await client()).coreSuperAdmin.grantInstanceAdmin({ personId }));
}

export async function revokeInstanceAdmin(personId: string): Promise<void> {
  return unwrap((await client()).coreSuperAdmin.revokeInstanceAdmin(personId));
}

export async function getAccountStatus(personId: string): Promise<AccountStatus> {
  return unwrap((await client()).coreSuperAdmin.getAccountStatus(personId));
}

export async function deactivateAccount(personId: string): Promise<AccountStatus> {
  return unwrap((await client()).coreSuperAdmin.deactivateAccount(personId));
}

export async function reactivateAccount(personId: string): Promise<AccountStatus> {
  return unwrap((await client()).coreSuperAdmin.reactivateAccount(personId));
}

export interface AuditLogFilter {
  actorPersonId?: string;
  targetKind?: string;
  targetId?: string;
  from?: string;
  to?: string;
  pageToken?: string;
}

/** M11.2 — keyset-paginated, filterable by actor/target/date; see components/data-table.tsx's own doc comment for why pagination state stays with the caller. */
export async function listAuditLog(filter: AuditLogFilter): Promise<AuditLogPage> {
  return unwrap(
    (await client()).coreSuperAdmin.listAuditLog(
      filter.actorPersonId,
      filter.targetKind,
      filter.targetId,
      filter.from,
      filter.to,
      undefined,
      filter.pageToken,
    ),
  );
}
