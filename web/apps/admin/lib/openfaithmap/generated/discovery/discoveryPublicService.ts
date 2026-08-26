import { IDiscoverySite } from "./discoverySite";
import { ISearchResult } from "./searchResult";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Anonymous public map/search (openfaithmap-web holds no session — D-AdminSurface). Never widens what internal/religion.SearchSites would already return publicly (the position-oracle fix: hidden sites excluded, others coordinate-snapped). See docs/modules/discovery.md.
 *
 */
export interface IDiscoveryPublicService {
    search(lat?: number | "NaN" | null, lng?: number | "NaN" | null, radiusM?: number | "NaN" | null, tradition?: string | null, language?: string | null, dayOfWeek?: number | null, query?: string | null): Promise<ISearchResult>;
    /**
     * A single congregation's discoverable site, always live (never the disposable search cache) — the per-congregation detail page's server-rendered fetch (M13.0). Throws SiteNotFound if the unit has no public, non-hidden site.
     *
     */
    getSite(unitId: string): Promise<IDiscoverySite>;
}

export class DiscoveryPublicService implements IDiscoveryPublicService {
    constructor(private bridge: IHttpApiBridge) {
    }

    public search(lat?: number | "NaN" | null, lng?: number | "NaN" | null, radiusM?: number | "NaN" | null, tradition?: string | null, language?: string | null, dayOfWeek?: number | null, query?: string | null): Promise<ISearchResult> {
        return this.bridge.call<ISearchResult>(
            "DiscoveryPublicService",
            "search",
            "GET",
            "/discovery/v1/search",
            __undefined,
            __undefined,
            {
                "lat": lat,
                "lng": lng,
                "radiusM": radiusM,
                "tradition": tradition,
                "language": language,
                "dayOfWeek": dayOfWeek,
                "query": query,
            },
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * A single congregation's discoverable site, always live (never the disposable search cache) — the per-congregation detail page's server-rendered fetch (M13.0). Throws SiteNotFound if the unit has no public, non-hidden site.
     *
     */
    public getSite(unitId: string): Promise<IDiscoverySite> {
        return this.bridge.call<IDiscoverySite>(
            "DiscoveryPublicService",
            "getSite",
            "GET",
            "/discovery/v1/sites/{unitId}",
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
}
