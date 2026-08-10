import { ICreateVouchRequest } from "./createVouchRequest";
import { IGuarantorStatusRecord } from "./guarantorStatusRecord";
import { IRevokeGuarantorRequest } from "./revokeGuarantorRequest";
import { IVouch } from "./vouch";
import { IVouchPage } from "./vouchPage";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Filing and reviewing vouches, and managing guarantor standing. See docs/modules/vouching.md.
 *
 */
export interface IVouchingService {
    /**
     * File a vouch as the caller (the guarantor). Rejects with Vouching:Forbidden if the caller lacks religionorg.manage on guarantorCongregationUnitId, and with Vouching:GuarantorRevoked if the caller currently holds REVOKED status — both checked live, never cached.
     *
     */
    createVouch(request: ICreateVouchRequest): Promise<IVouch>;
    /** List vouches for a claim. Requires platform-moderator standing. */
    listVouches(claimant?: string | null, congregation?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<IVouchPage>;
    /**
     * Revoke a guarantor. Requires platform-moderator standing. Never invalidates the guarantor's past vouches automatically — instead files one moderation report per affected vouch for moderator review (vouching.md's invariant).
     *
     */
    revokeGuarantor(personRid: string, request: IRevokeGuarantorRequest): Promise<IGuarantorStatusRecord>;
    /** Read a guarantor's current status. Requires platform-moderator standing. */
    getGuarantorStatus(personRid: string): Promise<IGuarantorStatusRecord>;
}

export class VouchingService implements IVouchingService {
    constructor(private bridge: IHttpApiBridge) {
    }

    /**
     * File a vouch as the caller (the guarantor). Rejects with Vouching:Forbidden if the caller lacks religionorg.manage on guarantorCongregationUnitId, and with Vouching:GuarantorRevoked if the caller currently holds REVOKED status — both checked live, never cached.
     *
     */
    public createVouch(request: ICreateVouchRequest): Promise<IVouch> {
        return this.bridge.call<IVouch>(
            "VouchingService",
            "createVouch",
            "POST",
            "/vouching/v1/vouches",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /** List vouches for a claim. Requires platform-moderator standing. */
    public listVouches(claimant?: string | null, congregation?: string | null, pageSize?: number | null, pageToken?: string | null): Promise<IVouchPage> {
        return this.bridge.call<IVouchPage>(
            "VouchingService",
            "listVouches",
            "GET",
            "/vouching/v1/vouches",
            __undefined,
            __undefined,
            {
                "claimant": claimant,
                "congregation": congregation,
                "pageSize": pageSize,
                "pageToken": pageToken,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * Revoke a guarantor. Requires platform-moderator standing. Never invalidates the guarantor's past vouches automatically — instead files one moderation report per affected vouch for moderator review (vouching.md's invariant).
     *
     */
    public revokeGuarantor(personRid: string, request: IRevokeGuarantorRequest): Promise<IGuarantorStatusRecord> {
        return this.bridge.call<IGuarantorStatusRecord>(
            "VouchingService",
            "revokeGuarantor",
            "POST",
            "/vouching/v1/guarantors/{personRid}/revoke",
            request,
            __undefined,
            __undefined,
            [
                personRid,
            ],
            __undefined,
            __undefined
        );
    }

    /** Read a guarantor's current status. Requires platform-moderator standing. */
    public getGuarantorStatus(personRid: string): Promise<IGuarantorStatusRecord> {
        return this.bridge.call<IGuarantorStatusRecord>(
            "VouchingService",
            "getGuarantorStatus",
            "GET",
            "/vouching/v1/guarantors/{personRid}/status",
            __undefined,
            __undefined,
            __undefined,
            [
                personRid,
            ],
            __undefined,
            __undefined
        );
    }
}
