import { IAccountStatus } from "./accountStatus";
import { IApiKeyPage } from "./apiKeyPage";
import { IAuditLogPage } from "./auditLogPage";
import { IBulkGrantUnitRoleRequest } from "./bulkGrantUnitRoleRequest";
import { IGrantInstanceAdminRequest } from "./grantInstanceAdminRequest";
import { IGrantUnitRoleRequest } from "./grantUnitRoleRequest";
import { IInstanceAdminGrant } from "./instanceAdminGrant";
import { IInstanceAdminPage } from "./instanceAdminPage";
import { IInvitePersonRequest } from "./invitePersonRequest";
import { IInviteResult } from "./inviteResult";
import { IMergePersonsRequest } from "./mergePersonsRequest";
import { IMergePreview } from "./mergePreview";
import { IMergeResult } from "./mergeResult";
import { IPersonPage } from "./personPage";
import { IRoleAssignmentPage } from "./roleAssignmentPage";
import { IRolePage } from "./rolePage";
import { ISessionPage } from "./sessionPage";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * The super-admin surface replacing the deleted oikumenea-console (D-SuperAdminFold): people search, role catalog, per-unit role-assignment grant/list/revoke, and the instance-admin plane's own grant/list/revoke. Every endpoint in this service is gated as a whole route group by internal/authz/transport.RequireInstanceAdmin, attached once at registration (cmd/openfaithmap-api/register_core.go) — not per-handler, so no future endpoint added here can be added without inheriting the check.
 *
 */
export interface ICoreSuperAdminService {
    searchPersons(query?: string | null, limit?: number | null): Promise<IPersonPage>;
    listRoles(): Promise<IRolePage>;
    listRoleAssignmentsByUnit(unitId: string): Promise<IRoleAssignmentPage>;
    grantUnitRole(request: IGrantUnitRoleRequest): Promise<void>;
    /**
     * M11.7 — grants roleId on unitId to every id in personIds, atomically, in one transaction. A fresh top-level resource (not nested under /role-assignments/) deliberately: M11.6's POST /persons/invite collided with an existing {personId} wildcard sibling and caused a boot-time httprouter radix-tree panic — /role-assignments/{assignmentId} already exists as a wildcard sibling here, so this avoids the same class of collision.
     *
     */
    bulkGrantUnitRole(request: IBulkGrantUnitRoleRequest): Promise<void>;
    revokeRoleAssignment(assignmentId: string): Promise<void>;
    listInstanceAdmins(): Promise<IInstanceAdminPage>;
    grantInstanceAdmin(request: IGrantInstanceAdminRequest): Promise<IInstanceAdminGrant>;
    revokeInstanceAdmin(personId: string): Promise<void>;
    /** M11.1 — D-AccountStatusEnforcement. */
    getAccountStatus(personId: string): Promise<IAccountStatus>;
    /** M11.1 — rejects further authentication for this person's account. Idempotent. */
    deactivateAccount(personId: string): Promise<IAccountStatus>;
    /** M11.1 — reverses deactivateAccount. Idempotent. */
    reactivateAccount(personId: string): Promise<IAccountStatus>;
    /**
     * M11.8 — read-only preview of what mergePersons would move/end for (personId as survivor, duplicatePersonId). Out of scope: registration/moderation/vouching/congregationimport rows, which reference person ids as opaque text with no FK.
     *
     */
    previewMergePersons(personId: string, request: IMergePersonsRequest): Promise<IMergePreview>;
    /**
     * M11.8 — reassigns duplicatePersonId's active role-assignment and membership rows onto personId (the survivor); moves the duplicate's account onto the survivor if the survivor has none, otherwise disables the duplicate's account (soft-merge — its login stops working); soft-deletes the duplicate person; audit-logs the merge. Destructive-shaped and irreversible. Out of scope: registration/moderation/vouching/congregationimport rows (opaque text, no FK) — those keep referencing the pre-merge duplicate id. The admin UI calls previewMergePersons first.
     *
     */
    mergePersons(personId: string, request: IMergePersonsRequest): Promise<IMergeResult>;
    /** M11.3 — personId's active sessions, admin-scoped. */
    listSessions(personId: string): Promise<ISessionPage>;
    /** M11.3 — revokes one of personId's sessions, admin-scoped. */
    revokeSession(personId: string, sessionId: string): Promise<void>;
    /**
     * M11.9 — personId's active AND revoked API keys, admin-scoped, metadata only (ApiKey carries no secret/hash field, so this endpoint can never leak one). Incident-response visibility: lets an admin see a person's keys without the owner's cooperation.
     *
     */
    listApiKeys(personId: string): Promise<IApiKeyPage>;
    /**
     * M11.9 — revokes one of personId's API keys, admin-scoped (incident response — kill a compromised key without waiting on the owner). Audit-logged distinctly from a self-revoke (REVOKE_API_KEY_ADMIN vs REVOKE_API_KEY) so the trail shows who actually acted.
     *
     */
    revokeApiKey(personId: string, apiKeyId: string): Promise<void>;
    /**
     * M11.2 — the shared logging helper's read side: every mutating super-admin action, keyset paginated (same real-pagination convention as Moderation's listReports/listAppeals, M7), filterable by actor/target/date, all filters ANDed together when set.
     *
     */
    listAuditLog(actorPersonId?: string | null, targetKind?: string | null, targetId?: string | null, from?: string | null, to?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<IAuditLogPage>;
    /**
     * M11.6, D-InviteLinkMVP — pre-provisions a Person+Account for the given email/displayName and returns a one-time invite token; the admin app builds the shareable link from its own origin. Must produce a row M10.2's existing JIT link-on-match logic will actually match on the invitee's first Google sign-in (IDENTITY_JIT_MATCH=account-email). A top-level /invites path, not nested under /persons/{personId}: unlike deactivate/reactivate, invite creation has no existing personId to path-parameter against — and httprouter's radix tree can't have a static "invite" segment as a sibling of the existing ":personId" wildcard under /persons/ anyway (a real boot-time panic caught by live-verifying this milestone).
     *
     */
    invitePerson(request: IInvitePersonRequest): Promise<IInviteResult>;
}

export class CoreSuperAdminService implements ICoreSuperAdminService {
    constructor(private bridge: IHttpApiBridge) {
    }

    public searchPersons(query?: string | null, limit?: number | null): Promise<IPersonPage> {
        return this.bridge.call<IPersonPage>(
            "CoreSuperAdminService",
            "searchPersons",
            "GET",
            "/core/v1/super-admin/persons",
            __undefined,
            __undefined,
            {
                "query": query,
                "limit": limit,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    public listRoles(): Promise<IRolePage> {
        return this.bridge.call<IRolePage>(
            "CoreSuperAdminService",
            "listRoles",
            "GET",
            "/core/v1/super-admin/roles",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    public listRoleAssignmentsByUnit(unitId: string): Promise<IRoleAssignmentPage> {
        return this.bridge.call<IRoleAssignmentPage>(
            "CoreSuperAdminService",
            "listRoleAssignmentsByUnit",
            "GET",
            "/core/v1/super-admin/units/{unitId}/role-assignments",
            __undefined,
            __undefined,
            __undefined,
            [
                unitId,
            ],
            __undefined,
            __undefined
        );
    }

    public grantUnitRole(request: IGrantUnitRoleRequest): Promise<void> {
        return this.bridge.call<void>(
            "CoreSuperAdminService",
            "grantUnitRole",
            "POST",
            "/core/v1/super-admin/role-assignments",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * M11.7 — grants roleId on unitId to every id in personIds, atomically, in one transaction. A fresh top-level resource (not nested under /role-assignments/) deliberately: M11.6's POST /persons/invite collided with an existing {personId} wildcard sibling and caused a boot-time httprouter radix-tree panic — /role-assignments/{assignmentId} already exists as a wildcard sibling here, so this avoids the same class of collision.
     *
     */
    public bulkGrantUnitRole(request: IBulkGrantUnitRoleRequest): Promise<void> {
        return this.bridge.call<void>(
            "CoreSuperAdminService",
            "bulkGrantUnitRole",
            "POST",
            "/core/v1/super-admin/bulk-role-assignments",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    public revokeRoleAssignment(assignmentId: string): Promise<void> {
        return this.bridge.call<void>(
            "CoreSuperAdminService",
            "revokeRoleAssignment",
            "DELETE",
            "/core/v1/super-admin/role-assignments/{assignmentId}",
            __undefined,
            __undefined,
            __undefined,
            [
                assignmentId,
            ],
            __undefined,
            __undefined
        );
    }

    public listInstanceAdmins(): Promise<IInstanceAdminPage> {
        return this.bridge.call<IInstanceAdminPage>(
            "CoreSuperAdminService",
            "listInstanceAdmins",
            "GET",
            "/core/v1/super-admin/instance-admins",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    public grantInstanceAdmin(request: IGrantInstanceAdminRequest): Promise<IInstanceAdminGrant> {
        return this.bridge.call<IInstanceAdminGrant>(
            "CoreSuperAdminService",
            "grantInstanceAdmin",
            "POST",
            "/core/v1/super-admin/instance-admins",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    public revokeInstanceAdmin(personId: string): Promise<void> {
        return this.bridge.call<void>(
            "CoreSuperAdminService",
            "revokeInstanceAdmin",
            "DELETE",
            "/core/v1/super-admin/instance-admins/{personId}",
            __undefined,
            __undefined,
            __undefined,
            [
                personId,
            ],
            __undefined,
            __undefined
        );
    }

    /** M11.1 — D-AccountStatusEnforcement. */
    public getAccountStatus(personId: string): Promise<IAccountStatus> {
        return this.bridge.call<IAccountStatus>(
            "CoreSuperAdminService",
            "getAccountStatus",
            "GET",
            "/core/v1/super-admin/persons/{personId}/account-status",
            __undefined,
            __undefined,
            __undefined,
            [
                personId,
            ],
            __undefined,
            __undefined
        );
    }

    /** M11.1 — rejects further authentication for this person's account. Idempotent. */
    public deactivateAccount(personId: string): Promise<IAccountStatus> {
        return this.bridge.call<IAccountStatus>(
            "CoreSuperAdminService",
            "deactivateAccount",
            "POST",
            "/core/v1/super-admin/persons/{personId}/deactivate",
            __undefined,
            __undefined,
            __undefined,
            [
                personId,
            ],
            __undefined,
            __undefined
        );
    }

    /** M11.1 — reverses deactivateAccount. Idempotent. */
    public reactivateAccount(personId: string): Promise<IAccountStatus> {
        return this.bridge.call<IAccountStatus>(
            "CoreSuperAdminService",
            "reactivateAccount",
            "POST",
            "/core/v1/super-admin/persons/{personId}/reactivate",
            __undefined,
            __undefined,
            __undefined,
            [
                personId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M11.8 — read-only preview of what mergePersons would move/end for (personId as survivor, duplicatePersonId). Out of scope: registration/moderation/vouching/congregationimport rows, which reference person ids as opaque text with no FK.
     *
     */
    public previewMergePersons(personId: string, request: IMergePersonsRequest): Promise<IMergePreview> {
        return this.bridge.call<IMergePreview>(
            "CoreSuperAdminService",
            "previewMergePersons",
            "POST",
            "/core/v1/super-admin/persons/{personId}/merge-preview",
            request,
            __undefined,
            __undefined,
            [
                personId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M11.8 — reassigns duplicatePersonId's active role-assignment and membership rows onto personId (the survivor); moves the duplicate's account onto the survivor if the survivor has none, otherwise disables the duplicate's account (soft-merge — its login stops working); soft-deletes the duplicate person; audit-logs the merge. Destructive-shaped and irreversible. Out of scope: registration/moderation/vouching/congregationimport rows (opaque text, no FK) — those keep referencing the pre-merge duplicate id. The admin UI calls previewMergePersons first.
     *
     */
    public mergePersons(personId: string, request: IMergePersonsRequest): Promise<IMergeResult> {
        return this.bridge.call<IMergeResult>(
            "CoreSuperAdminService",
            "mergePersons",
            "POST",
            "/core/v1/super-admin/persons/{personId}/merge",
            request,
            __undefined,
            __undefined,
            [
                personId,
            ],
            __undefined,
            __undefined
        );
    }

    /** M11.3 — personId's active sessions, admin-scoped. */
    public listSessions(personId: string): Promise<ISessionPage> {
        return this.bridge.call<ISessionPage>(
            "CoreSuperAdminService",
            "listSessions",
            "GET",
            "/core/v1/super-admin/persons/{personId}/sessions",
            __undefined,
            __undefined,
            __undefined,
            [
                personId,
            ],
            __undefined,
            __undefined
        );
    }

    /** M11.3 — revokes one of personId's sessions, admin-scoped. */
    public revokeSession(personId: string, sessionId: string): Promise<void> {
        return this.bridge.call<void>(
            "CoreSuperAdminService",
            "revokeSession",
            "DELETE",
            "/core/v1/super-admin/persons/{personId}/sessions/{sessionId}",
            __undefined,
            __undefined,
            __undefined,
            [
                personId,
                sessionId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M11.9 — personId's active AND revoked API keys, admin-scoped, metadata only (ApiKey carries no secret/hash field, so this endpoint can never leak one). Incident-response visibility: lets an admin see a person's keys without the owner's cooperation.
     *
     */
    public listApiKeys(personId: string): Promise<IApiKeyPage> {
        return this.bridge.call<IApiKeyPage>(
            "CoreSuperAdminService",
            "listApiKeys",
            "GET",
            "/core/v1/super-admin/persons/{personId}/api-keys",
            __undefined,
            __undefined,
            __undefined,
            [
                personId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M11.9 — revokes one of personId's API keys, admin-scoped (incident response — kill a compromised key without waiting on the owner). Audit-logged distinctly from a self-revoke (REVOKE_API_KEY_ADMIN vs REVOKE_API_KEY) so the trail shows who actually acted.
     *
     */
    public revokeApiKey(personId: string, apiKeyId: string): Promise<void> {
        return this.bridge.call<void>(
            "CoreSuperAdminService",
            "revokeApiKey",
            "DELETE",
            "/core/v1/super-admin/persons/{personId}/api-keys/{apiKeyId}",
            __undefined,
            __undefined,
            __undefined,
            [
                personId,
                apiKeyId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M11.2 — the shared logging helper's read side: every mutating super-admin action, keyset paginated (same real-pagination convention as Moderation's listReports/listAppeals, M7), filterable by actor/target/date, all filters ANDed together when set.
     *
     */
    public listAuditLog(actorPersonId?: string | null, targetKind?: string | null, targetId?: string | null, from?: string | null, to?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<IAuditLogPage> {
        return this.bridge.call<IAuditLogPage>(
            "CoreSuperAdminService",
            "listAuditLog",
            "GET",
            "/core/v1/super-admin/audit-log",
            __undefined,
            __undefined,
            {
                "actorPersonId": actorPersonId,
                "targetKind": targetKind,
                "targetId": targetId,
                "from": from,
                "to": to,
                "pageSize": pageSize,
                "pageToken": pageToken,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * M11.6, D-InviteLinkMVP — pre-provisions a Person+Account for the given email/displayName and returns a one-time invite token; the admin app builds the shareable link from its own origin. Must produce a row M10.2's existing JIT link-on-match logic will actually match on the invitee's first Google sign-in (IDENTITY_JIT_MATCH=account-email). A top-level /invites path, not nested under /persons/{personId}: unlike deactivate/reactivate, invite creation has no existing personId to path-parameter against — and httprouter's radix tree can't have a static "invite" segment as a sibling of the existing ":personId" wildcard under /persons/ anyway (a real boot-time panic caught by live-verifying this milestone).
     *
     */
    public invitePerson(request: IInvitePersonRequest): Promise<IInviteResult> {
        return this.bridge.call<IInviteResult>(
            "CoreSuperAdminService",
            "invitePerson",
            "POST",
            "/core/v1/super-admin/invites",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }
}
