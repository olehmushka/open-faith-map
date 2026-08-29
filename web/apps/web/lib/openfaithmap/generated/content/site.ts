import { ISocialLinks } from "./socialLinks";

export interface ISite {
    'id': string;
    /** The directory_units RID this site belongs to (opaque value). */
    'congregationUnitId': string;
    'slug': string;
    /** Accent color, font pairing, header layout — data, never a per-tenant code fork. */
    'theme': any;
    /** M14.11. content_sites' own setting — never a content document. */
    'logoUrl'?: string | null;
    /** M14.11. content_sites' own setting — never a content document. */
    'socialLinks': ISocialLinks;
    'createdAt': string;
    'updatedAt': string;
}
