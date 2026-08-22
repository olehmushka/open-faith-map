import { IInviteInfo } from "./inviteInfo";
import { IResolveInviteRequest } from "./resolveInviteRequest";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * M11.6 — genuinely anonymous, mirroring ContentPublicService's own shape (no default-auth: a Conjure service's auth is a fixed per-service choice — see this file's own header comment — so an endpoint reachable with no bearer at all cannot live on CoreService, which sets default-auth: header). The invitee has no session yet (they haven't signed in for the first time), the same reasoning D-AdminSurface gives for ContentPublicService, just admin-side instead of web-side. internal/identity/middleware's isBypassPath also carries a matching /core/v1/public prefix bypass, the same mechanism /content/v1/public already uses.
 *
 */
export interface ICorePublicService {
    /**
     * Validates an invite token for its own not-yet-authenticated invitee, ahead of their first sign-in.
     *
     */
    resolveInvite(request: IResolveInviteRequest): Promise<IInviteInfo>;
}

export class CorePublicService implements ICorePublicService {
    constructor(private bridge: IHttpApiBridge) {
    }

    /**
     * Validates an invite token for its own not-yet-authenticated invitee, ahead of their first sign-in.
     *
     */
    public resolveInvite(request: IResolveInviteRequest): Promise<IInviteInfo> {
        return this.bridge.call<IInviteInfo>(
            "CorePublicService",
            "resolveInvite",
            "POST",
            "/core/v1/public/invites/resolve",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }
}
