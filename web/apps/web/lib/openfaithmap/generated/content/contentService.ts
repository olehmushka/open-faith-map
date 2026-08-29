import { IBlockList } from "./blockList";
import { ICreateDocumentRequest } from "./createDocumentRequest";
import { ICreateSiteRequest } from "./createSiteRequest";
import { IDocument } from "./document";
import { IDocumentPage } from "./documentPage";
import { INavItemList } from "./navItemList";
import { IPreviewLink } from "./previewLink";
import { IPutBlocksRequest } from "./putBlocksRequest";
import { IPutNavItemsRequest } from "./putNavItemsRequest";
import { IRevisionPage } from "./revisionPage";
import { ISite } from "./site";
import { ITransitionDocumentRequest } from "./transitionDocumentRequest";
import { IUpdateDocumentRequest } from "./updateDocumentRequest";
import { IUpdateSiteThemeRequest } from "./updateSiteThemeRequest";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Congregation site authoring: sites, documents (pages), blocks. Every write, and every read that must see draft state, is content.manage-gated — a live, target-scoped Authorize call against the site's own congregation unit (see file header). See docs/modules/content.md.
 *
 */
export interface IContentService {
    createSite(request: ICreateSiteRequest): Promise<ISite>;
    updateSiteTheme(siteId: string, request: IUpdateSiteThemeRequest): Promise<ISite>;
    /** Admin read — returns documents in every state. */
    listDocuments(siteId: string, kind?: string | null, locale?: string | null, state?: string | null): Promise<IDocumentPage>;
    createDocument(siteId: string, request: ICreateDocumentRequest): Promise<IDocument>;
    updateDocument(documentId: string, request: IUpdateDocumentRequest): Promise<IDocument>;
    transitionDocument(documentId: string, request: ITransitionDocumentRequest): Promise<IDocument>;
    /** Admin read — works regardless of document state. */
    getBlocks(documentId: string): Promise<IBlockList>;
    /**
     * Full replace of the draft revision's blocks, validated against each referenced block type's json_schema (M14.6: this writes the draft only — never what's published — so this one endpoint serves both the editor's manual save and its debounced autosave).
     *
     */
    putBlocks(documentId: string, request: IPutBlocksRequest): Promise<IBlockList>;
    /** History list (M14.6) — every past checkpoint, newest first, excluding the draft itself. */
    listRevisions(documentId: string): Promise<IRevisionPage>;
    /**
     * Copies a past checkpoint's blocks into the draft — into the draft only, never auto-publishing (owner decision, 2026-08-28). Publish afterward to make it live.
     *
     */
    restoreRevision(documentId: string, revisionId: string): Promise<IBlockList>;
    /**
     * M14.7. Mints a short-lived, site-scoped preview token — content.manage-gated, same as every other write/draft-read on this service. The returned token is handed to ContentPublicService's preview endpoints on the tenant subdomain, never used here again.
     *
     */
    createPreviewLink(siteId: string): Promise<IPreviewLink>;
    /** M14.10. Admin read of the site's nav menu, in sortOrder. */
    listNavItems(siteId: string): Promise<INavItemList>;
    /**
     * M14.10. Full replace of the site's nav menu — a small, hand-curated list edited as a batch, the same shape putBlocks used before M14.6's revision refactor moved it to an in-place update. Content:DuplicateNavItemSortOrder if two items share a sortOrder, Content:NavTargetAmbiguous if an item has neither or both of targetDocumentId/targetUrl, Content:NavTargetInvalid if targetDocumentId doesn't resolve to a PAGE document in this same site.
     *
     */
    putNavItems(siteId: string, request: IPutNavItemsRequest): Promise<INavItemList>;
}

export class ContentService implements IContentService {
    constructor(private bridge: IHttpApiBridge) {
    }

    public createSite(request: ICreateSiteRequest): Promise<ISite> {
        return this.bridge.call<ISite>(
            "ContentService",
            "createSite",
            "POST",
            "/content/v1/sites",
            request,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    public updateSiteTheme(siteId: string, request: IUpdateSiteThemeRequest): Promise<ISite> {
        return this.bridge.call<ISite>(
            "ContentService",
            "updateSiteTheme",
            "PUT",
            "/content/v1/sites/{siteId}/theme",
            request,
            __undefined,
            __undefined,
            [
                siteId,
            ],
            __undefined,
            __undefined
        );
    }

    /** Admin read — returns documents in every state. */
    public listDocuments(siteId: string, kind?: string | null, locale?: string | null, state?: string | null): Promise<IDocumentPage> {
        return this.bridge.call<IDocumentPage>(
            "ContentService",
            "listDocuments",
            "GET",
            "/content/v1/sites/{siteId}/documents",
            __undefined,
            __undefined,
            {
                "kind": kind,
                "locale": locale,
                "state": state,
            },
            [
                siteId,
            ],
            __undefined,
            __undefined
        );
    }

    public createDocument(siteId: string, request: ICreateDocumentRequest): Promise<IDocument> {
        return this.bridge.call<IDocument>(
            "ContentService",
            "createDocument",
            "POST",
            "/content/v1/sites/{siteId}/documents",
            request,
            __undefined,
            __undefined,
            [
                siteId,
            ],
            __undefined,
            __undefined
        );
    }

    public updateDocument(documentId: string, request: IUpdateDocumentRequest): Promise<IDocument> {
        return this.bridge.call<IDocument>(
            "ContentService",
            "updateDocument",
            "PUT",
            "/content/v1/documents/{documentId}",
            request,
            __undefined,
            __undefined,
            [
                documentId,
            ],
            __undefined,
            __undefined
        );
    }

    public transitionDocument(documentId: string, request: ITransitionDocumentRequest): Promise<IDocument> {
        return this.bridge.call<IDocument>(
            "ContentService",
            "transitionDocument",
            "POST",
            "/content/v1/documents/{documentId}/transition",
            request,
            __undefined,
            __undefined,
            [
                documentId,
            ],
            __undefined,
            __undefined
        );
    }

    /** Admin read — works regardless of document state. */
    public getBlocks(documentId: string): Promise<IBlockList> {
        return this.bridge.call<IBlockList>(
            "ContentService",
            "getBlocks",
            "GET",
            "/content/v1/documents/{documentId}/blocks",
            __undefined,
            __undefined,
            __undefined,
            [
                documentId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * Full replace of the draft revision's blocks, validated against each referenced block type's json_schema (M14.6: this writes the draft only — never what's published — so this one endpoint serves both the editor's manual save and its debounced autosave).
     *
     */
    public putBlocks(documentId: string, request: IPutBlocksRequest): Promise<IBlockList> {
        return this.bridge.call<IBlockList>(
            "ContentService",
            "putBlocks",
            "PUT",
            "/content/v1/documents/{documentId}/blocks",
            request,
            __undefined,
            __undefined,
            [
                documentId,
            ],
            __undefined,
            __undefined
        );
    }

    /** History list (M14.6) — every past checkpoint, newest first, excluding the draft itself. */
    public listRevisions(documentId: string): Promise<IRevisionPage> {
        return this.bridge.call<IRevisionPage>(
            "ContentService",
            "listRevisions",
            "GET",
            "/content/v1/documents/{documentId}/revisions",
            __undefined,
            __undefined,
            __undefined,
            [
                documentId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * Copies a past checkpoint's blocks into the draft — into the draft only, never auto-publishing (owner decision, 2026-08-28). Publish afterward to make it live.
     *
     */
    public restoreRevision(documentId: string, revisionId: string): Promise<IBlockList> {
        return this.bridge.call<IBlockList>(
            "ContentService",
            "restoreRevision",
            "POST",
            "/content/v1/documents/{documentId}/revisions/{revisionId}/restore",
            __undefined,
            __undefined,
            __undefined,
            [
                documentId,
                revisionId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M14.7. Mints a short-lived, site-scoped preview token — content.manage-gated, same as every other write/draft-read on this service. The returned token is handed to ContentPublicService's preview endpoints on the tenant subdomain, never used here again.
     *
     */
    public createPreviewLink(siteId: string): Promise<IPreviewLink> {
        return this.bridge.call<IPreviewLink>(
            "ContentService",
            "createPreviewLink",
            "POST",
            "/content/v1/sites/{siteId}/preview-link",
            __undefined,
            __undefined,
            __undefined,
            [
                siteId,
            ],
            __undefined,
            __undefined
        );
    }

    /** M14.10. Admin read of the site's nav menu, in sortOrder. */
    public listNavItems(siteId: string): Promise<INavItemList> {
        return this.bridge.call<INavItemList>(
            "ContentService",
            "listNavItems",
            "GET",
            "/content/v1/sites/{siteId}/nav-items",
            __undefined,
            __undefined,
            __undefined,
            [
                siteId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M14.10. Full replace of the site's nav menu — a small, hand-curated list edited as a batch, the same shape putBlocks used before M14.6's revision refactor moved it to an in-place update. Content:DuplicateNavItemSortOrder if two items share a sortOrder, Content:NavTargetAmbiguous if an item has neither or both of targetDocumentId/targetUrl, Content:NavTargetInvalid if targetDocumentId doesn't resolve to a PAGE document in this same site.
     *
     */
    public putNavItems(siteId: string, request: IPutNavItemsRequest): Promise<INavItemList> {
        return this.bridge.call<INavItemList>(
            "ContentService",
            "putNavItems",
            "PUT",
            "/content/v1/sites/{siteId}/nav-items",
            request,
            __undefined,
            __undefined,
            [
                siteId,
            ],
            __undefined,
            __undefined
        );
    }
}
