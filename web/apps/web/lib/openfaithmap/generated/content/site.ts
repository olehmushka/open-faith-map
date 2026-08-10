export interface ISite {
    'id': string;
    /** The go-oikumenea Unit RID this site belongs to (opaque foreign value). */
    'congregationUnitId': string;
    'slug': string;
    /** Accent color, font pairing, header layout — data, never a per-tenant code fork. */
    'theme': any;
    'createdAt': string;
    'updatedAt': string;
}
