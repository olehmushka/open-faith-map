import { IBlockList } from "./blockList";
import { IBlockTypePage } from "./blockTypePage";
import { IDocumentPage } from "./documentPage";
import { ISite } from "./site";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Anonymous reads only (openfaithmap-web holds no session — D-AdminSurface). Always filters to published/unlisted; never discloses draft documents or their blocks.
 *
 */
export interface IContentPublicService {
    getSite(congregationUnitId: string): Promise<ISite>;
    listPublicDocuments(siteId: string, kind?: string | null, locale?: string | null): Promise<IDocumentPage>;
    /** Content:DocumentNotFound if the document is draft or doesn't exist — never distinguishes the two. */
    getPublicBlocks(documentId: string): Promise<IBlockList>;
    /** Active block types only. */
    listBlockTypes(): Promise<IBlockTypePage>;
}

export class ContentPublicService implements IContentPublicService {
    constructor(private bridge: IHttpApiBridge) {
    }

    public getSite(congregationUnitId: string): Promise<ISite> {
        return this.bridge.call<ISite>(
            "ContentPublicService",
            "getSite",
            "GET",
            "/content/v1/public/units/{congregationUnitId}/site",
            __undefined,
            __undefined,
            __undefined,
            [
                congregationUnitId,
            ],
            __undefined,
            __undefined
        );
    }

    public listPublicDocuments(siteId: string, kind?: string | null, locale?: string | null): Promise<IDocumentPage> {
        return this.bridge.call<IDocumentPage>(
            "ContentPublicService",
            "listPublicDocuments",
            "GET",
            "/content/v1/public/sites/{siteId}/documents",
            __undefined,
            __undefined,
            {
                "kind": kind,
                "locale": locale,
            },
            [
                siteId,
            ],
            __undefined,
            __undefined
        );
    }

    /** Content:DocumentNotFound if the document is draft or doesn't exist — never distinguishes the two. */
    public getPublicBlocks(documentId: string): Promise<IBlockList> {
        return this.bridge.call<IBlockList>(
            "ContentPublicService",
            "getPublicBlocks",
            "GET",
            "/content/v1/public/documents/{documentId}/blocks",
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

    /** Active block types only. */
    public listBlockTypes(): Promise<IBlockTypePage> {
        return this.bridge.call<IBlockTypePage>(
            "ContentPublicService",
            "listBlockTypes",
            "GET",
            "/content/v1/public/block-types",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }
}
