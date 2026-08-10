import { IBlockList } from "./blockList";
import { ICreateDocumentRequest } from "./createDocumentRequest";
import { ICreateSiteRequest } from "./createSiteRequest";
import { IDocument } from "./document";
import { IDocumentPage } from "./documentPage";
import { IPutBlocksRequest } from "./putBlocksRequest";
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
    /** Full replace, validated against each referenced block type's json_schema. */
    putBlocks(documentId: string, request: IPutBlocksRequest): Promise<IBlockList>;
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

    /** Full replace, validated against each referenced block type's json_schema. */
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
}
