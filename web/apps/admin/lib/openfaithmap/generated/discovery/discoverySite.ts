export interface IDiscoverySite {
    'id': string;
    /** The go-oikumenea religion_sites RID this row projects (opaque foreign value). */
    'religionSiteRid': string;
    'congregationUnitRid': string;
    /** Set once the congregation has published a site (content module) — nullable. */
    'contentSiteId'?: string | null;
    /** Already public_precision-coarsened by go-oikumenea; null when precision is hidden. */
    'latitude'?: number | "NaN" | null;
    'longitude'?: number | "NaN" | null;
    'traditionTaxonId'?: string | null;
    'serviceLanguages': Array<string>;
    /** 0=Sunday .. 6=Saturday, matching go-oikumenea's ServiceSchedule.dayOfWeek. */
    'serviceDays': Array<number>;
    'refreshedAt': string;
}
