import { ICountryPage } from "./countryPage";
import { ICreateChildOrgRequest } from "./createChildOrgRequest";
import { IGetPersonsRequest } from "./getPersonsRequest";
import { IMembershipPage } from "./membershipPage";
import { IOrgKindPage } from "./orgKindPage";
import { IOrgProfile } from "./orgProfile";
import { IPerson } from "./person";
import { IPersonPage } from "./personPage";
import { IRegisterSessionRequest } from "./registerSessionRequest";
import { ISession } from "./session";
import { ISessionPage } from "./sessionPage";
import { ITaxon } from "./taxon";
import { ITaxonPage } from "./taxonPage";
import { IUnit } from "./unit";
import { IUnitPage } from "./unitPage";
import { IUnitRefPage } from "./unitRefPage";
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
