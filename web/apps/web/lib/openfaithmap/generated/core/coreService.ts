import { IApiKeyPage } from "./apiKeyPage";
import { ICountryPage } from "./countryPage";
import { ICreateApiKeyRequest } from "./createApiKeyRequest";
import { ICreateApiKeyResult } from "./createApiKeyResult";
import { ICreateChildOrgRequest } from "./createChildOrgRequest";
import { ICreateUnitRequest } from "./createUnitRequest";
import { IGetPersonsRequest } from "./getPersonsRequest";
import { IMembershipPage } from "./membershipPage";
import { IMoveUnitRequest } from "./moveUnitRequest";
import { IOrgKindPage } from "./orgKindPage";
import { IOrgProfile } from "./orgProfile";
import { IPermissionCodePage } from "./permissionCodePage";
import { IPerson } from "./person";
import { IPersonPage } from "./personPage";
import { IRegisterSessionRequest } from "./registerSessionRequest";
import { IRoleAssignmentPage } from "./roleAssignmentPage";
import { ISession } from "./session";
import { ISessionPage } from "./sessionPage";
import { ISetUnitStateRequest } from "./setUnitStateRequest";
import { ITaxon } from "./taxon";
import { ITaxonPage } from "./taxonPage";
import { IUnit } from "./unit";
import { IUnitMoveJob } from "./unitMoveJob";
import { IUnitPage } from "./unitPage";
import { IUnitRefPage } from "./unitRefPage";
import { IUpdateMyProfileRequest } from "./updateMyProfileRequest";
import { IUpdateUnitRequest } from "./updateUnitRequest";
import { IWhoami } from "./whoami";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * The admin app's session-gated reads over the in-process core (units, taxa, countries, org kinds/profiles, memberships, persons) plus its one gated write, createChildOrg. See file header for exactly which endpoints carry an authorization gate beyond the session itself.
 *
 */
export interface ICoreService {
    whoami(): Promise<IWhoami>;
    /**
     * M11.3 — creates the identity_sessions row backing a just-completed NextAuth sign-in. Exempt from the per-request session-id check every other endpoint now requires (internal/identity/middleware's sessionExemptRoutes) — this is what creates that row, so it cannot itself require one to already exist.
     *
     */
    registerSession(request: IRegisterSessionRequest): Promise<ISession>;
    /** M11.3 — the caller's own active sessions, self-scoped. */
    listMySessions(): Promise<ISessionPage>;
    /** M11.3 — revokes one of the caller's own sessions, self-scoped. */
    revokeMySession(sessionId: string): Promise<void>;
    /** M11.5 — updates the caller's own display name, self-scoped. */
    updateMyProfile(request: IUpdateMyProfileRequest): Promise<IPerson>;
    /** M11.5 — the caller's own active role assignments across every unit, self-scoped. */
    listMyRoleAssignments(): Promise<IRoleAssignmentPage>;
    /**
     * M11.9 — the closed unit-scoped permission catalog, self-scoped since every person needs it for their own createApiKey allowlist picker, not just admins. Static, no DB read.
     *
     */
    listPermissionCatalog(): Promise<IPermissionCodePage>;
    /** M11.9 — the caller's own active API keys, self-scoped. */
    listMyApiKeys(): Promise<IApiKeyPage>;
    /**
     * M11.9 — mints a new API key for the caller, scoped to request.permissionCodes. Returns the raw secret exactly once.
     *
     */
    createApiKey(request: ICreateApiKeyRequest): Promise<ICreateApiKeyResult>;
    /** M11.9 — revokes one of the caller's own API keys, self-scoped. */
    revokeMyApiKey(apiKeyId: string): Promise<void>;
    getUnit(unitId: string): Promise<IUnit>;
    /** Free-text search over code/name, capped at limit (default/max 50). */
    listUnits(query?: string | null, limit?: number | null): Promise<IUnitPage>;
    unitAncestors(unitId: string): Promise<IUnitRefPage>;
    /** Free-text search over code/name, capped at limit (default/max 50). */
    listTaxa(query?: string | null, limit?: number | null): Promise<ITaxonPage>;
    getTaxon(taxonId: string): Promise<ITaxon>;
    listOrgKinds(): Promise<IOrgKindPage>;
    getOrgProfile(unitId: string): Promise<IOrgProfile>;
    /** Gated — the caller must hold religionorg.manage over parentUnitId. */
    createChildOrg(request: ICreateChildOrgRequest): Promise<IOrgProfile>;
    /** M12.1 — general unit creation under a parent. Gated — the caller must hold unit.lifecycle over request.parentUnitId. */
    createUnit(request: ICreateUnitRequest): Promise<IUnit>;
    /**
     * M12.1 — rewrites name/code/level. Gated — the caller must hold unit.lifecycle over unitId. Deliberately not nested under /units/: httprouter (vendored v1.3.0) cannot register a wildcard child next to the existing static /units/children (createChildOrg) at the same POST tree depth — any route of the shape POST /units/{unitId}/* panics at startup regardless of what follows the wildcard, since a POST /units/children request would otherwise be genuinely ambiguous between the two routes. A sibling top-level resource, not a suffix, is the only fix; grantUnitRole/bulkGrantUnitRole already set this precedent (POST /role-assignments, not POST /units/{unitId}/role-assignments).
     *
     */
    updateUnit(unitId: string, request: IUpdateUnitRequest): Promise<IUnit>;
    /**
     * M12.1 — archive/suspend/reactivate. Gated — the caller must hold unit.lifecycle over unitId. The root unit refuses every state change. See updateUnit's docs for why this lives under /unit-lifecycle/ rather than /units/.
     *
     */
    setUnitState(unitId: string, request: ISetUnitStateRequest): Promise<IUnit>;
    /**
     * M12.1 — soft-delete. Gated — the caller must hold unit.lifecycle over unitId. Refuses the root unit outright, and is orphan-protected against child units, active role assignments, and an existing religion org profile.
     *
     */
    deleteUnit(unitId: string): Promise<void>;
    /**
     * M12.2 — starts or resumes moving unitId onto request.newParentUnitId, generalized out of internal/registration's own former private reparent state machine. Gated — D-UnitMoveDualScope: the caller must hold unit.edges.manage on BOTH unitId's current parent and request.newParentUnitId. Add-before-remove (unitId briefly has two parents mid-move, never zero) and resumable — a repeat call while a live job targets the same new parent resumes it; targeting a different parent while one is live is a conflict (unitMoveConflict). A sibling top-level resource, not nested under /units/, for the same httprouter reason updateUnit/ setUnitState already are (see updateUnit's own docs).
     *
     */
    moveUnit(unitId: string, request: IMoveUnitRequest): Promise<IUnitMoveJob>;
    /** Read the most recent move job for unitId (any graph — see graphCode), if one has ever been started. */
    getUnitMoveStatus(unitId: string, graphCode?: string | null): Promise<IUnitMoveJob | null>;
    listCountries(): Promise<ICountryPage>;
    listMembershipsByUnit(unitId: string): Promise<IMembershipPage>;
    getPerson(personId: string): Promise<IPerson>;
    /** Batched read — replaces the pre-cutover my-congregation page's per-member getPerson loop. */
    getPersons(request: IGetPersonsRequest): Promise<IPersonPage>;
}

export class CoreService implements ICoreService {
    constructor(private bridge: IHttpApiBridge) {
    }

    public whoami(): Promise<IWhoami> {
        return this.bridge.call<IWhoami>(
            "CoreService",
            "whoami",
            "GET",
            "/core/v1/whoami",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * M11.3 — creates the identity_sessions row backing a just-completed NextAuth sign-in. Exempt from the per-request session-id check every other endpoint now requires (internal/identity/middleware's sessionExemptRoutes) — this is what creates that row, so it cannot itself require one to already exist.
     *
     */
    public registerSession(request: IRegisterSessionRequest): Promise<ISession> {
        return this.bridge.call<ISession>(
            "CoreService",
            "registerSession",
            "POST",
            "/core/v1/sessions",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /** M11.3 — the caller's own active sessions, self-scoped. */
    public listMySessions(): Promise<ISessionPage> {
        return this.bridge.call<ISessionPage>(
            "CoreService",
            "listMySessions",
            "GET",
            "/core/v1/sessions",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /** M11.3 — revokes one of the caller's own sessions, self-scoped. */
    public revokeMySession(sessionId: string): Promise<void> {
        return this.bridge.call<void>(
            "CoreService",
            "revokeMySession",
            "DELETE",
            "/core/v1/sessions/{sessionId}",
            __undefined,
            __undefined,
            __undefined,
            [
                sessionId,
            ],
            __undefined,
            __undefined
        );
    }

    /** M11.5 — updates the caller's own display name, self-scoped. */
    public updateMyProfile(request: IUpdateMyProfileRequest): Promise<IPerson> {
        return this.bridge.call<IPerson>(
            "CoreService",
            "updateMyProfile",
            "PUT",
            "/core/v1/profile",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /** M11.5 — the caller's own active role assignments across every unit, self-scoped. */
    public listMyRoleAssignments(): Promise<IRoleAssignmentPage> {
        return this.bridge.call<IRoleAssignmentPage>(
            "CoreService",
            "listMyRoleAssignments",
            "GET",
            "/core/v1/profile/roles",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * M11.9 — the closed unit-scoped permission catalog, self-scoped since every person needs it for their own createApiKey allowlist picker, not just admins. Static, no DB read.
     *
     */
    public listPermissionCatalog(): Promise<IPermissionCodePage> {
        return this.bridge.call<IPermissionCodePage>(
            "CoreService",
            "listPermissionCatalog",
            "GET",
            "/core/v1/permission-catalog",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /** M11.9 — the caller's own active API keys, self-scoped. */
    public listMyApiKeys(): Promise<IApiKeyPage> {
        return this.bridge.call<IApiKeyPage>(
            "CoreService",
            "listMyApiKeys",
            "GET",
            "/core/v1/api-keys",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * M11.9 — mints a new API key for the caller, scoped to request.permissionCodes. Returns the raw secret exactly once.
     *
     */
    public createApiKey(request: ICreateApiKeyRequest): Promise<ICreateApiKeyResult> {
        return this.bridge.call<ICreateApiKeyResult>(
            "CoreService",
            "createApiKey",
            "POST",
            "/core/v1/api-keys",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /** M11.9 — revokes one of the caller's own API keys, self-scoped. */
    public revokeMyApiKey(apiKeyId: string): Promise<void> {
        return this.bridge.call<void>(
            "CoreService",
            "revokeMyApiKey",
            "DELETE",
            "/core/v1/api-keys/{apiKeyId}",
            __undefined,
            __undefined,
            __undefined,
            [
                apiKeyId,
            ],
            __undefined,
            __undefined
        );
    }

    public getUnit(unitId: string): Promise<IUnit> {
        return this.bridge.call<IUnit>(
            "CoreService",
            "getUnit",
            "GET",
            "/core/v1/units/{unitId}",
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

    /** Free-text search over code/name, capped at limit (default/max 50). */
    public listUnits(query?: string | null, limit?: number | null): Promise<IUnitPage> {
        return this.bridge.call<IUnitPage>(
            "CoreService",
            "listUnits",
            "GET",
            "/core/v1/units",
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

    public unitAncestors(unitId: string): Promise<IUnitRefPage> {
        return this.bridge.call<IUnitRefPage>(
            "CoreService",
            "unitAncestors",
            "GET",
            "/core/v1/units/{unitId}/ancestors",
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

    /** Free-text search over code/name, capped at limit (default/max 50). */
    public listTaxa(query?: string | null, limit?: number | null): Promise<ITaxonPage> {
        return this.bridge.call<ITaxonPage>(
            "CoreService",
            "listTaxa",
            "GET",
            "/core/v1/taxa",
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

    public getTaxon(taxonId: string): Promise<ITaxon> {
        return this.bridge.call<ITaxon>(
            "CoreService",
            "getTaxon",
            "GET",
            "/core/v1/taxa/{taxonId}",
            __undefined,
            __undefined,
            __undefined,
            [
                taxonId,
            ],
            __undefined,
            __undefined
        );
    }

    public listOrgKinds(): Promise<IOrgKindPage> {
        return this.bridge.call<IOrgKindPage>(
            "CoreService",
            "listOrgKinds",
            "GET",
            "/core/v1/org-kinds",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    public getOrgProfile(unitId: string): Promise<IOrgProfile> {
        return this.bridge.call<IOrgProfile>(
            "CoreService",
            "getOrgProfile",
            "GET",
            "/core/v1/units/{unitId}/org-profile",
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

    /** Gated — the caller must hold religionorg.manage over parentUnitId. */
    public createChildOrg(request: ICreateChildOrgRequest): Promise<IOrgProfile> {
        return this.bridge.call<IOrgProfile>(
            "CoreService",
            "createChildOrg",
            "POST",
            "/core/v1/units/children",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /** M12.1 — general unit creation under a parent. Gated — the caller must hold unit.lifecycle over request.parentUnitId. */
    public createUnit(request: ICreateUnitRequest): Promise<IUnit> {
        return this.bridge.call<IUnit>(
            "CoreService",
            "createUnit",
            "POST",
            "/core/v1/units",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * M12.1 — rewrites name/code/level. Gated — the caller must hold unit.lifecycle over unitId. Deliberately not nested under /units/: httprouter (vendored v1.3.0) cannot register a wildcard child next to the existing static /units/children (createChildOrg) at the same POST tree depth — any route of the shape POST /units/{unitId}/* panics at startup regardless of what follows the wildcard, since a POST /units/children request would otherwise be genuinely ambiguous between the two routes. A sibling top-level resource, not a suffix, is the only fix; grantUnitRole/bulkGrantUnitRole already set this precedent (POST /role-assignments, not POST /units/{unitId}/role-assignments).
     *
     */
    public updateUnit(unitId: string, request: IUpdateUnitRequest): Promise<IUnit> {
        return this.bridge.call<IUnit>(
            "CoreService",
            "updateUnit",
            "POST",
            "/core/v1/unit-lifecycle/{unitId}",
            request,
            __undefined,
            __undefined,
            [
                unitId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M12.1 — archive/suspend/reactivate. Gated — the caller must hold unit.lifecycle over unitId. The root unit refuses every state change. See updateUnit's docs for why this lives under /unit-lifecycle/ rather than /units/.
     *
     */
    public setUnitState(unitId: string, request: ISetUnitStateRequest): Promise<IUnit> {
        return this.bridge.call<IUnit>(
            "CoreService",
            "setUnitState",
            "POST",
            "/core/v1/unit-lifecycle/{unitId}/state",
            request,
            __undefined,
            __undefined,
            [
                unitId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M12.1 — soft-delete. Gated — the caller must hold unit.lifecycle over unitId. Refuses the root unit outright, and is orphan-protected against child units, active role assignments, and an existing religion org profile.
     *
     */
    public deleteUnit(unitId: string): Promise<void> {
        return this.bridge.call<void>(
            "CoreService",
            "deleteUnit",
            "DELETE",
            "/core/v1/units/{unitId}",
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

    /**
     * M12.2 — starts or resumes moving unitId onto request.newParentUnitId, generalized out of internal/registration's own former private reparent state machine. Gated — D-UnitMoveDualScope: the caller must hold unit.edges.manage on BOTH unitId's current parent and request.newParentUnitId. Add-before-remove (unitId briefly has two parents mid-move, never zero) and resumable — a repeat call while a live job targets the same new parent resumes it; targeting a different parent while one is live is a conflict (unitMoveConflict). A sibling top-level resource, not nested under /units/, for the same httprouter reason updateUnit/ setUnitState already are (see updateUnit's own docs).
     *
     */
    public moveUnit(unitId: string, request: IMoveUnitRequest): Promise<IUnitMoveJob> {
        return this.bridge.call<IUnitMoveJob>(
            "CoreService",
            "moveUnit",
            "POST",
            "/core/v1/unit-moves/{unitId}",
            request,
            __undefined,
            __undefined,
            [
                unitId,
            ],
            __undefined,
            __undefined
        );
    }

    /** Read the most recent move job for unitId (any graph — see graphCode), if one has ever been started. */
    public getUnitMoveStatus(unitId: string, graphCode?: string | null): Promise<IUnitMoveJob | null> {
        return this.bridge.call<IUnitMoveJob | null>(
            "CoreService",
            "getUnitMoveStatus",
            "GET",
            "/core/v1/unit-moves/{unitId}",
            __undefined,
            __undefined,
            {
                "graphCode": graphCode,
            },
            [
                unitId,
            ],
            __undefined,
            __undefined
        );
    }

    public listCountries(): Promise<ICountryPage> {
        return this.bridge.call<ICountryPage>(
            "CoreService",
            "listCountries",
            "GET",
            "/core/v1/countries",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    public listMembershipsByUnit(unitId: string): Promise<IMembershipPage> {
        return this.bridge.call<IMembershipPage>(
            "CoreService",
            "listMembershipsByUnit",
            "GET",
            "/core/v1/units/{unitId}/memberships",
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

    public getPerson(personId: string): Promise<IPerson> {
        return this.bridge.call<IPerson>(
            "CoreService",
            "getPerson",
            "GET",
            "/core/v1/persons/{personId}",
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

    /** Batched read — replaces the pre-cutover my-congregation page's per-member getPerson loop. */
    public getPersons(request: IGetPersonsRequest): Promise<IPersonPage> {
        return this.bridge.call<IPersonPage>(
            "CoreService",
            "getPersons",
            "POST",
            "/core/v1/persons/batch-get",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }
}
