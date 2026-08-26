import { ISite } from "./site";
import { IUpdateSiteAttributesRequest } from "./updateSiteAttributesRequest";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Authenticated congregation-admin/registration-operator surface over religion_sites — the physical/online presence a discovery search or a congregation's own admin form reads. site.manage-gated, target-scoped to each unit. See docs/modules/religion.md.
 *
 */
export interface IReligionService {
    /**
     * The unit's primary site, exact/uncoarsened (an owner's own private view, unlike discovery's public-precision-filtered projection). Throws SiteNotFound if the unit has no site yet.
     *
     */
    getSite(unitId: string): Promise<ISite>;
    /**
     * Overwrites the unit's primary site's attributes wholesale (the admin form always submits the complete SiteAttributes shape, never a partial patch). Throws SiteNotFound if the unit has no site yet.
     *
     */
    updateSiteAttributes(unitId: string, request: IUpdateSiteAttributesRequest): Promise<ISite>;
}

export class ReligionService implements IReligionService {
    constructor(private bridge: IHttpApiBridge) {
    }

    /**
     * The unit's primary site, exact/uncoarsened (an owner's own private view, unlike discovery's public-precision-filtered projection). Throws SiteNotFound if the unit has no site yet.
     *
     */
    public getSite(unitId: string): Promise<ISite> {
        return this.bridge.call<ISite>(
            "ReligionService",
            "getSite",
            "GET",
            "/religion/v1/units/{unitId}/site",
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
     * Overwrites the unit's primary site's attributes wholesale (the admin form always submits the complete SiteAttributes shape, never a partial patch). Throws SiteNotFound if the unit has no site yet.
     *
     */
    public updateSiteAttributes(unitId: string, request: IUpdateSiteAttributesRequest): Promise<ISite> {
        return this.bridge.call<ISite>(
            "ReligionService",
            "updateSiteAttributes",
            "PUT",
            "/religion/v1/units/{unitId}/site/attributes",
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
}
