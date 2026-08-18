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
    'traditionTaxonId'?: string | null;
    'serviceLanguages': Array<string>;
    /** 0=Sunday .. 6=Saturday, matching religion_service_schedules.day_of_week. */
    'serviceDays': Array<number>;
    'refreshedAt': string;
}
