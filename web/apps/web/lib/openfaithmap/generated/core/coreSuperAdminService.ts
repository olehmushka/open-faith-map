import { IAccountStatus } from "./accountStatus";
import { IAuditLogPage } from "./auditLogPage";
import { IGrantInstanceAdminRequest } from "./grantInstanceAdminRequest";
import { IGrantUnitRoleRequest } from "./grantUnitRoleRequest";
import { IInstanceAdminGrant } from "./instanceAdminGrant";
import { IInstanceAdminPage } from "./instanceAdminPage";
import { IPersonPage } from "./personPage";
import { IRoleAssignmentPage } from "./roleAssignmentPage";
import { IRolePage } from "./rolePage";
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
     * M11.2 — the shared logging helper's read side: every mutating super-admin action, keyset paginated (same real-pagination convention as Moderation's listReports/listAppeals, M7), filterable by actor/target/date, all filters ANDed together when set.
     *
     */
    listAuditLog(actorPersonId?: string | null, targetKind?: string | null, targetId?: string | null, from?: string | null, to?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<IAuditLogPage>;
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
}
