import { IRefreshRegionRequest } from "./refreshRegionRequest";
import { IRefreshResult } from "./refreshResult";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Operator tooling only — forcing a manual cache rebuild for a region. Not part of the public product surface; the public map never calls this.
 *
 */
export interface IDiscoveryService {
    refresh(request: IRefreshRegionRequest): Promise<IRefreshResult>;
}

export class DiscoveryService implements IDiscoveryService {
    constructor(private bridge: IHttpApiBridge) {
    }

    public refresh(request: IRefreshRegionRequest): Promise<IRefreshResult> {
        return this.bridge.call<IRefreshResult>(
            "DiscoveryService",
            "refresh",
            "POST",
            "/discovery/v1/refresh",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }
}
