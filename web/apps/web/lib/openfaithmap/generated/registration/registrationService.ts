import { IApproveRegistrationRequest } from "./approveRegistrationRequest";
import { IRegistrationRequest } from "./registrationRequest";
import { IRegistrationRequestPage } from "./registrationRequestPage";
import { IRejectRegistrationRequest } from "./rejectRegistrationRequest";
import { ISubmitRegistrationRequest } from "./submitRegistrationRequest";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Congregation-registration requests: submit (any authenticated person), list/approve/reject (a registration operator — verified live against go-oikumenea's PDP, not a locally-cached role). See docs/modules/registration.md.
 *
 */
export interface IRegistrationService {
    /**
     * Submit a new registration request as the caller. Runs the D-Exclusions taxon check (walking the taxon's ancestors via go-oikumenea's religion.read) before persisting — returns Registration:TaxonExcluded if the tradition (or an ancestor) is on the named exclusion list, Registration:TaxonNotFound if taxonId doesn't resolve.
     *
     */
    submitRequest(request: ISubmitRegistrationRequest): Promise<IRegistrationRequest>;
    /**
     * List registration requests. Operator-only for now (verified live: does the caller hold religionorg.manage on the configured root unit?) — see open seams in docs/modules/registration.md for a submitter's-own-requests view.
     *
     */
    listRequests(status?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<IRegistrationRequestPage>;
    /** Read one request. The submitter or an operator (verified live) may read it. */
    getRequest(requestId: string): Promise<IRegistrationRequest>;
    /**
     * Approve a PENDING request: performs the real go-oikumenea writes (createChildOrg under the configured root unit, org classification, a site over a new location, a filled Position, and a unit-scoped role assignment granting the submitter authority over their new congregation) using the caller's own forwarded token — go-oikumenea's PDP decides for real; this returns whatever error go-oikumenea's PDP does if the caller lacks authority.
     *
     */
    approveRequest(requestId: string, request: IApproveRegistrationRequest): Promise<IRegistrationRequest>;
    /** Reject a PENDING request with a reason. No go-oikumenea writes. */
    rejectRequest(requestId: string, request: IRejectRegistrationRequest): Promise<IRegistrationRequest>;
}

export class RegistrationService implements IRegistrationService {
    constructor(private bridge: IHttpApiBridge) {
    }

    /**
     * Submit a new registration request as the caller. Runs the D-Exclusions taxon check (walking the taxon's ancestors via go-oikumenea's religion.read) before persisting — returns Registration:TaxonExcluded if the tradition (or an ancestor) is on the named exclusion list, Registration:TaxonNotFound if taxonId doesn't resolve.
     *
     */
    public submitRequest(request: ISubmitRegistrationRequest): Promise<IRegistrationRequest> {
        return this.bridge.call<IRegistrationRequest>(
            "RegistrationService",
            "submitRequest",
            "POST",
            "/registration/v1/requests",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * List registration requests. Operator-only for now (verified live: does the caller hold religionorg.manage on the configured root unit?) — see open seams in docs/modules/registration.md for a submitter's-own-requests view.
     *
     */
    public listRequests(status?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<IRegistrationRequestPage> {
        return this.bridge.call<IRegistrationRequestPage>(
            "RegistrationService",
            "listRequests",
            "GET",
            "/registration/v1/requests",
            __undefined,
            __undefined,
            {
                "status": status,
                "pageSize": pageSize,
                "pageToken": pageToken,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    /** Read one request. The submitter or an operator (verified live) may read it. */
    public getRequest(requestId: string): Promise<IRegistrationRequest> {
        return this.bridge.call<IRegistrationRequest>(
            "RegistrationService",
            "getRequest",
            "GET",
            "/registration/v1/requests/{requestId}",
            __undefined,
            __undefined,
            __undefined,
            [
                requestId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * Approve a PENDING request: performs the real go-oikumenea writes (createChildOrg under the configured root unit, org classification, a site over a new location, a filled Position, and a unit-scoped role assignment granting the submitter authority over their new congregation) using the caller's own forwarded token — go-oikumenea's PDP decides for real; this returns whatever error go-oikumenea's PDP does if the caller lacks authority.
     *
     */
    public approveRequest(requestId: string, request: IApproveRegistrationRequest): Promise<IRegistrationRequest> {
        return this.bridge.call<IRegistrationRequest>(
            "RegistrationService",
            "approveRequest",
            "POST",
            "/registration/v1/requests/{requestId}/approve",
            request,
            __undefined,
            __undefined,
            [
                requestId,
            ],
            __undefined,
            __undefined
        );
    }

    /** Reject a PENDING request with a reason. No go-oikumenea writes. */
    public rejectRequest(requestId: string, request: IRejectRegistrationRequest): Promise<IRegistrationRequest> {
        return this.bridge.call<IRegistrationRequest>(
            "RegistrationService",
            "rejectRequest",
            "POST",
            "/registration/v1/requests/{requestId}/reject",
            request,
            __undefined,
            __undefined,
            [
                requestId,
            ],
            __undefined,
            __undefined
        );
    }
}
