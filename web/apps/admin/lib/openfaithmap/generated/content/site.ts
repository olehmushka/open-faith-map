export interface ISite {
    'id': string;
    /** The directory_units RID this site belongs to (opaque value). */
    'congregationUnitId': string;
    'slug': string;
    /** Accent color, font pairing, header layout — data, never a per-tenant code fork. */
    'theme': any;
    'createdAt': string;
    'updatedAt': string;
}
