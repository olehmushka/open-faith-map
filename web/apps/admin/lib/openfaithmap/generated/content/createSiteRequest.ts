export interface ICreateSiteRequest {
    'congregationUnitId': string;
    /** Admin-chosen, probed for uniqueness at write time (U5) — Content:SlugTaken on collision. */
    'slug': string;
}
