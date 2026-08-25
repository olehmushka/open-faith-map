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
  IAccessExplanation,
  IAccessExplanationContribution,
  IAccountStatus,
  IApiKey,
  IAuditLogEntry,
  IAuditLogPage,
  ICountry,
  ICreateApiKeyResult,
  ICreateChildOrgRequest,
  ICreateUnitRequest,
  IInstanceAdminGrant,
  IInviteInfo,
  IInviteResult,
  IMembership,
  IMergePreview,
  IMergeResult,
  IOrgKind,
  IOrgProfile,
  IPerson,
  IRole,
  IRoleAssignment,
  ISession,
  ITaxon,
  IUnit,
  IUnitDeleteEligibility,
  IUnitRef,
  IUpdateUnitRequest,
  IWhoami,
} from "./openfaithmap/generated/core";

export type Whoami = IWhoami;
export type AccessExplanation = IAccessExplanation;
export type AccessExplanationContribution = IAccessExplanationContribution;
export type Unit = IUnit;
export type UnitRef = IUnitRef;
export type UnitDeleteEligibility = IUnitDeleteEligibility;
export type CreateUnitInput = ICreateUnitRequest;
export type UpdateUnitInput = IUpdateUnitRequest;
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
export type Session = ISession;
export type InviteResult = IInviteResult;
export type InviteInfo = IInviteInfo;
export type MergePreview = IMergePreview;
export type MergeResult = IMergeResult;
export type ApiKey = IApiKey;
export type CreateApiKeyResult = ICreateApiKeyResult;

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
    // M11.3, D-SessionTracking: the bearer above is Google's own signed ID token, which can't carry
    // a custom sessionId claim — session.sessionId (stamped by auth.ts's jwt() callback at sign-in)
    // travels instead as its own header, alongside the bearer, checked by the identity middleware's
    // per-request session lookup (internal/identity/middleware.Authenticator.Handle).
    fetch: session?.sessionId
      ? (url, init) =>
          fetch(url, { ...init, headers: { ...init?.headers, "X-Session-Id": session.sessionId! } })
      : undefined,
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

// M12.5 — unit lifecycle CRUD wrappers over M12.1's endpoints. Gated server-side — the caller must
// hold unit.lifecycle over the relevant unit (parentUnitId for createUnit, unitId otherwise), same as
// createChildOrg above.
export async function createUnit(input: CreateUnitInput): Promise<Unit> {
  return unwrap((await client()).core.createUnit(input));
}

export async function updateUnit(unitId: string, input: UpdateUnitInput): Promise<Unit> {
  return unwrap((await client()).core.updateUnit(unitId, input));
}

export async function setUnitState(unitId: string, state: string): Promise<Unit> {
  return unwrap((await client()).core.setUnitState(unitId, { state }));
}

export async function deleteUnit(unitId: string): Promise<void> {
  return unwrap((await client()).core.deleteUnit(unitId));
}

/** Gated server-side — no more permissive than deleteUnit itself (M12.5). */
export async function unitDeleteEligibility(unitId: string): Promise<UnitDeleteEligibility> {
  return unwrap((await client()).core.unitDeleteEligibility(unitId));
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

/**
 * M11.3 — the caller's own active sessions, self-scoped. registerSession itself has no wrapper
 * here — it's called directly from auth.ts's jwt() callback, before a session exists to build this
 * file's own client() with (see registerSessionOnBackend there).
 */
export async function listMySessions(): Promise<Session[]> {
  const page = await unwrap((await client()).core.listMySessions());
  return page.sessions;
}

/** M11.3 — revokes one of the caller's own sessions, self-scoped. */
export async function revokeMySession(sessionId: string): Promise<void> {
  return unwrap((await client()).core.revokeMySession(sessionId));
}

/** M11.5 — updates the caller's own display name, self-scoped. */
export async function updateMyProfile(displayName: string): Promise<Person> {
  return unwrap((await client()).core.updateMyProfile({ displayName }));
}

/** M11.5 — the caller's own active role assignments across every unit, self-scoped. */
export async function listMyRoleAssignments(): Promise<RoleAssignment[]> {
  const page = await unwrap((await client()).core.listMyRoleAssignments());
  return page.assignments;
}

/**
 * M11.6 — validates an invite token for its own not-yet-authenticated invitee. Deliberately its own
 * unauthenticated client, not client(): this is called from the public /accept-invite page, where
 * there is no NextAuth session (and so no idToken/X-Session-Id) to forward — the backend's own
 * anonymousRoutes allowlist (internal/identity/middleware) is what actually makes this call work
 * with no bearer at all.
 */
export async function resolveInvite(token: string): Promise<InviteInfo> {
  const anonymousClient = createOpenFaithMapClient({ baseUrl: requireBaseUrl() });
  return unwrap(anonymousClient.corePublic.resolveInvite({ token }));
}

/** M11.9 — the closed unit-scoped permission catalog, self-scoped (every person needs it for their own createApiKey picker). */
export async function listPermissionCatalog(): Promise<string[]> {
  const page = await unwrap((await client()).core.listPermissionCatalog());
  return page.codes;
}

/** M11.9 — the caller's own active API keys, self-scoped. */
export async function listMyApiKeys(): Promise<ApiKey[]> {
  const page = await unwrap((await client()).core.listMyApiKeys());
  return page.apiKeys;
}

/** M11.9 — mints a new API key for the caller, scoped to permissionCodes. token is returned exactly once. */
export async function createApiKey(label: string, permissionCodes: string[]): Promise<CreateApiKeyResult> {
  return unwrap((await client()).core.createApiKey({ label, permissionCodes }));
}

/** M11.9 — revokes one of the caller's own API keys, self-scoped. */
export async function revokeMyApiKey(apiKeyId: string): Promise<void> {
  return unwrap((await client()).core.revokeMyApiKey(apiKeyId));
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

// M12.4 — decision-tracing debug tool ("why does this user have this access"); pure read, no
// audit log entry on the server side.
export async function explainAccess(
  subjectPersonId: string,
  permissionCode: string,
  unitId: string,
): Promise<AccessExplanation> {
  return unwrap((await client()).coreSuperAdmin.explainAccess(subjectPersonId, permissionCode, unitId));
}

// expiresAt (M12.3) is an optional ISO-8601 datetime string — nil/omitted for a non-expiring grant.
// scope is hardcoded "unit" (M12.2 added real scope="subtree" provisioning, but no UI in this app
// collects it yet — this wrapper preserves the exact behavior every grant from this screen already
// had before the generated SDK caught up to the contract).
export async function grantUnitRole(
  personId: string,
  roleId: string,
  unitId: string,
  expiresAt?: string,
): Promise<void> {
  return unwrap((await client()).coreSuperAdmin.grantUnitRole({ personId, roleId, unitId, scope: "unit", expiresAt }));
}

export async function revokeRoleAssignment(assignmentId: string): Promise<void> {
  return unwrap((await client()).coreSuperAdmin.revokeRoleAssignment(assignmentId));
}

// M12.3 — clears an active assignment's expiresAt, leaving the grant itself untouched.
export async function clearRoleAssignmentExpiry(assignmentId: string): Promise<void> {
  return unwrap((await client()).coreSuperAdmin.clearRoleAssignmentExpiry(assignmentId));
}

// M11.7 — the batch variant of grantUnitRole: the same role and unit, granted to every id in
// personIds at once, atomically. expiresAt (M12.3) follows grantUnitRole's own rule; scope is
// hardcoded "unit" for the same reason grantUnitRole's own wrapper is.
export async function bulkGrantUnitRole(
  personIds: string[],
  roleId: string,
  unitId: string,
  expiresAt?: string,
): Promise<void> {
  return unwrap(
    (await client()).coreSuperAdmin.bulkGrantUnitRole({ personIds, roleId, unitId, scope: "unit", expiresAt }),
  );
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

/** M11.8 — read-only preview of what mergePersons(personId, duplicatePersonId) would move/end. */
export async function previewMergePersons(personId: string, duplicatePersonId: string): Promise<MergePreview> {
  return unwrap((await client()).coreSuperAdmin.previewMergePersons(personId, { duplicatePersonId }));
}

/**
 * M11.8 — merges duplicatePersonId into personId (the survivor): reassigns its active role
 * assignments and memberships, moves or disables its account, soft-deletes it. Destructive and
 * irreversible; callers should call previewMergePersons first.
 */
export async function mergePersons(personId: string, duplicatePersonId: string): Promise<MergeResult> {
  return unwrap((await client()).coreSuperAdmin.mergePersons(personId, { duplicatePersonId }));
}

/** M11.3 — personId's active sessions, admin-scoped. */
export async function listSessions(personId: string): Promise<Session[]> {
  const page = await unwrap((await client()).coreSuperAdmin.listSessions(personId));
  return page.sessions;
}

/** M11.3 — revokes one of personId's sessions, admin-scoped. */
export async function revokeSession(personId: string, sessionId: string): Promise<void> {
  return unwrap((await client()).coreSuperAdmin.revokeSession(personId, sessionId));
}

/** M11.9 — personId's full API key history (active and revoked), admin-scoped incident-response visibility. */
export async function listApiKeys(personId: string): Promise<ApiKey[]> {
  const page = await unwrap((await client()).coreSuperAdmin.listApiKeys(personId));
  return page.apiKeys;
}

/** M11.9 — revokes one of personId's API keys, admin-scoped (incident response). */
export async function revokeApiKey(personId: string, apiKeyId: string): Promise<void> {
  return unwrap((await client()).coreSuperAdmin.revokeApiKey(personId, apiKeyId));
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

/** M11.6, D-InviteLinkMVP — pre-provisions a Person+Account and returns a one-time invite token; the caller builds the shareable link from its own known origin. */
export async function invitePerson(email: string, displayName: string): Promise<InviteResult> {
  return unwrap((await client()).coreSuperAdmin.invitePerson({ email, displayName }));
}
