import { IBlockInput } from "./blockInput";

/**
 * M14.13, D-SitePatterns. A pre-built starting layout — inserting one copies blocks into a document and detaches immediately (unsynced: no ongoing link back to this row). Reused both by the moderator's catalog page (create/update/delete) and by the document editor's own insert-a-pattern UI, which reads blocks directly and appends them client-side.
 *
 */
export interface IPattern {
    'id': string;
    'name': string;
    'description': string;
    'blocks': Array<IBlockInput>;
    'sortOrder': number;
    'createdAt': string;
    'updatedAt': string;
}
