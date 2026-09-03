import { IBlockList } from "./blockList";
import { IBlockTypePage } from "./blockTypePage";
import { IDocumentPage } from "./documentPage";
import { IDocumentWithAncestors } from "./documentWithAncestors";
import { IPatternPage } from "./patternPage";
import { IPublicNavItemList } from "./publicNavItemList";
import { ISite } from "./site";
import { ISiteChrome } from "./siteChrome";
import { ISubmitContactFormRequest } from "./submitContactFormRequest";
import type { IHttpApiBridge } from "conjure-client";

/** Constant reference to `undefined` that we expect to get minified and therefore reduce total code size */
const __undefined: undefined = undefined;

/**
 * Anonymous reads only (openfaithmap-web holds no session — D-AdminSurface). Always filters to published/unlisted; never discloses draft documents or their blocks — with exactly one carve-out, M14.7's listPreviewDocuments/getPreviewBlocks, which require a valid site-scoped preview token (minted by ContentService.createPreviewLink) in place of a session, since this service's caller never holds one.
 *
 */
export interface IContentPublicService {
    getSite(congregationUnitId: string): Promise<ISite>;
    /**
     * M14.9: the tenant-subdomain proxy resolves a Host header's slug through this endpoint. A distinct top-level path (not nested under /sites/{siteId}/...) — same httprouter wildcard-slot conflict getSite's own comment above documents.
     *
     */
    getSiteBySlug(slug: string): Promise<ISite>;
    /**
     * M14.11. The tenant layout's one call for header/footer data — logoUrl/socialLinks from content_sites, congregationName/address/schedules composed live from religion at read time. No auth, same anonymous shape as getSite/getSiteBySlug.
     *
     */
    getSiteChrome(siteId: string): Promise<ISiteChrome>;
    listPublicDocuments(siteId: string, kind?: string | null, locale?: string | null): Promise<IDocumentPage>;
    /** Content:DocumentNotFound if the document is draft or doesn't exist — never distinguishes the two. */
    getPublicBlocks(documentId: string): Promise<IBlockList>;
    /**
     * M14.7. Like listPublicDocuments, but returns documents in every state (draft included) — gated by token (from createPreviewLink) instead of published/unlisted filtering. Content:PreviewTokenInvalid if the token is missing, malformed, expired, or scoped to a different site.
     *
     */
    listPreviewDocuments(siteId: string, token: string, kind?: string | null, locale?: string | null): Promise<IDocumentPage>;
    /**
     * M14.7. Reads the document's draft revision regardless of its published state — gated by token (from createPreviewLink) instead of a session. Content:PreviewTokenInvalid if the token is missing, malformed, expired, or scoped to a different site than the document's own.
     *
     */
    getPreviewBlocks(documentId: string, token: string): Promise<IBlockList>;
    /** Active block types only. */
    listBlockTypes(): Promise<IBlockTypePage>;
    /**
     * M14.13. Not sensitive data (same reasoning listBlockTypes already uses for having no auth) — every pattern, in sortOrder. The document editor's insert-a-pattern UI calls this exact endpoint to fetch blocks to copy client-side, the same way it already calls listBlockTypes.
     *
     */
    listPatterns(): Promise<IPatternPage>;
    /**
     * M14.10. Resolved hrefs, in sortOrder — see PublicNavItem's own docs for the omit-on-missing-or-draft-target behavior.
     *
     */
    listPublicNavItems(siteId: string): Promise<IPublicNavItemList>;
    /**
     * M14.10. Resolves the leaf PAGE document (by locale + slug) plus its real ancestor chain, for the tenant-subdomain catch-all page route. path is a slash-joined, ordered list of slug segments (e.g. "parent-slug/child-slug"); every segment must match the document's real parent_document_id chain positionally — a mismatch at any position (including the leaf's own slug) 404s exactly like a wrong slug would, never resolving by the last segment alone. Content:DocumentNotFound if the leaf doesn't exist, isn't a PAGE, is DRAFT, or the ancestor chain doesn't match — one error for every case, same discipline as getPublicBlocks.
     *
     */
    getPublicDocumentByPath(siteId: string, locale: string, path: string): Promise<IDocumentWithAncestors>;
    /**
     * M14.16, D-InAppInbox. Genuinely anonymous — the third such write in the codebase, after moderation's two. Rate-limited (internal/platform/ratelimit, wrapping this whole service's registration — see cmd/openfaithmap-api/register_content.go). Always succeeds for a honeypot-triggered or too-fast submission; only Content:FormSubmissionInvalid (empty message) and Content:SiteNotFound are ever returned as errors.
     *
     */
    submitContactForm(siteId: string, request: ISubmitContactFormRequest): Promise<void>;
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

    /**
     * M14.9: the tenant-subdomain proxy resolves a Host header's slug through this endpoint. A distinct top-level path (not nested under /sites/{siteId}/...) — same httprouter wildcard-slot conflict getSite's own comment above documents.
     *
     */
    public getSiteBySlug(slug: string): Promise<ISite> {
        return this.bridge.call<ISite>(
            "ContentPublicService",
            "getSiteBySlug",
            "GET",
            "/content/v1/public/site-by-slug/{slug}",
            __undefined,
            __undefined,
            __undefined,
            [
                slug,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M14.11. The tenant layout's one call for header/footer data — logoUrl/socialLinks from content_sites, congregationName/address/schedules composed live from religion at read time. No auth, same anonymous shape as getSite/getSiteBySlug.
     *
     */
    public getSiteChrome(siteId: string): Promise<ISiteChrome> {
        return this.bridge.call<ISiteChrome>(
            "ContentPublicService",
            "getSiteChrome",
            "GET",
            "/content/v1/public/sites/{siteId}/chrome",
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

    /**
     * M14.7. Like listPublicDocuments, but returns documents in every state (draft included) — gated by token (from createPreviewLink) instead of published/unlisted filtering. Content:PreviewTokenInvalid if the token is missing, malformed, expired, or scoped to a different site.
     *
     */
    public listPreviewDocuments(siteId: string, token: string, kind?: string | null, locale?: string | null): Promise<IDocumentPage> {
        return this.bridge.call<IDocumentPage>(
            "ContentPublicService",
            "listPreviewDocuments",
            "GET",
            "/content/v1/public/sites/{siteId}/preview-documents",
            __undefined,
            __undefined,
            {
                "token": token,
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

    /**
     * M14.7. Reads the document's draft revision regardless of its published state — gated by token (from createPreviewLink) instead of a session. Content:PreviewTokenInvalid if the token is missing, malformed, expired, or scoped to a different site than the document's own.
     *
     */
    public getPreviewBlocks(documentId: string, token: string): Promise<IBlockList> {
        return this.bridge.call<IBlockList>(
            "ContentPublicService",
            "getPreviewBlocks",
            "GET",
            "/content/v1/public/documents/{documentId}/preview-blocks",
            __undefined,
            __undefined,
            {
                "token": token,
            },
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

    /**
     * M14.13. Not sensitive data (same reasoning listBlockTypes already uses for having no auth) — every pattern, in sortOrder. The document editor's insert-a-pattern UI calls this exact endpoint to fetch blocks to copy client-side, the same way it already calls listBlockTypes.
     *
     */
    public listPatterns(): Promise<IPatternPage> {
        return this.bridge.call<IPatternPage>(
            "ContentPublicService",
            "listPatterns",
            "GET",
            "/content/v1/public/patterns",
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined,
            __undefined
        );
    }

    /**
     * M14.10. Resolved hrefs, in sortOrder — see PublicNavItem's own docs for the omit-on-missing-or-draft-target behavior.
     *
     */
    public listPublicNavItems(siteId: string): Promise<IPublicNavItemList> {
        return this.bridge.call<IPublicNavItemList>(
            "ContentPublicService",
            "listPublicNavItems",
            "GET",
            "/content/v1/public/sites/{siteId}/nav-items",
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
     * M14.10. Resolves the leaf PAGE document (by locale + slug) plus its real ancestor chain, for the tenant-subdomain catch-all page route. path is a slash-joined, ordered list of slug segments (e.g. "parent-slug/child-slug"); every segment must match the document's real parent_document_id chain positionally — a mismatch at any position (including the leaf's own slug) 404s exactly like a wrong slug would, never resolving by the last segment alone. Content:DocumentNotFound if the leaf doesn't exist, isn't a PAGE, is DRAFT, or the ancestor chain doesn't match — one error for every case, same discipline as getPublicBlocks.
     *
     */
    public getPublicDocumentByPath(siteId: string, locale: string, path: string): Promise<IDocumentWithAncestors> {
        return this.bridge.call<IDocumentWithAncestors>(
            "ContentPublicService",
            "getPublicDocumentByPath",
            "GET",
            "/content/v1/public/sites/{siteId}/documents/by-path",
            __undefined,
            __undefined,
            {
                "locale": locale,
                "path": path,
            },
            [
                siteId,
            ],
            __undefined,
            __undefined
        );
    }

    /**
     * M14.16, D-InAppInbox. Genuinely anonymous — the third such write in the codebase, after moderation's two. Rate-limited (internal/platform/ratelimit, wrapping this whole service's registration — see cmd/openfaithmap-api/register_content.go). Always succeeds for a honeypot-triggered or too-fast submission; only Content:FormSubmissionInvalid (empty message) and Content:SiteNotFound are ever returned as errors.
     *
     */
    public submitContactForm(siteId: string, request: ISubmitContactFormRequest): Promise<void> {
        return this.bridge.call<void>(
            "ContentPublicService",
            "submitContactForm",
            "POST",
            "/content/v1/public/sites/{siteId}/contact",
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
