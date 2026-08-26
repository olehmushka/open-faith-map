import { ISiteAttributes } from "./siteAttributes";

export interface IDiscoverySite {
    'id': string;
    /** The religion_sites RID this row projects (opaque value). */
    'religionSiteRid': string;
    'congregationUnitRid': string;
    /** Set once the congregation has published a site (content module) — nullable. */
    'contentSiteId'?: string | null;
    /** Already public_precision-coarsened by internal/religion.Coarsen; null when precision is hidden. */
    'latitude'?: number | "NaN" | null;
    'longitude'?: number | "NaN" | null;
    /** The congregation's display name (directory_units.name) — shown regardless of public_precision (D-DiscoveryAddressPrecision). */
    'name': string;
    /**
     * Precision-coarsened address text (D-DiscoveryAddressPrecision) — full street address at exact/street, locality-only at neighborhood/city, absent at hidden.
     *
     */
    'address'?: string | null;
    'traditionTaxonId'?: string | null;
    'traditionTaxonCode'?: string | null;
    'traditionTaxonName'?: string | null;
    'serviceLanguages': Array<string>;
    /** 0=Sunday .. 6=Saturday, matching religion_service_schedules.day_of_week. */
    'serviceDays': Array<number>;
    'attributes': ISiteAttributes;
    'refreshedAt': string;
}
