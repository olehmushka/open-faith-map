// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// openfaithmap-api's content module public reads via the generated TypeScript SDK (M4). No
// session, ever (D-AdminSurface) — ContentPublicService never asks for a token. Mirrors
// web/apps/admin's lib/content.ts public half, minus the content.manage-gated admin functions,
// which this app can never call.
import { isConjureError } from "conjure-client";
import { cache } from "react";

import { createOpenFaithMapClient, type FetchFunction } from "./openfaithmap";
import type { IBlock, IBlockType, IDocument, IDocumentTranslation, IDocumentWithAncestors, IPublicNavItem, ISite, ISiteChrome, ISitemapEntry, ISubmitContactFormRequest } from "./openfaithmap/generated/content";

export type Site = ISite;
export type Document = IDocument;
export type Block = IBlock;
export type BlockType = IBlockType;
export type PublicNavItem = IPublicNavItem;
export type DocumentWithAncestors = IDocumentWithAncestors;
export type DocumentTranslation = IDocumentTranslation;
export type SiteChrome = ISiteChrome;
export type SitemapEntry = ISitemapEntry;
export type SubmitContactFormInput = ISubmitContactFormRequest;

export class ContentApiError extends Error {
  constructor(
    public status: number,
    public errorName: string,
    public parameters: Record<string, unknown>,
  ) {
    super(`${errorName} (${status})`);
  }
}

function requireBaseUrl(): string {
  const raw = process.env.OPENFAITHMAP_API_BASE_URL?.trim();
  if (!raw) {
    throw new Error("OPENFAITHMAP_API_BASE_URL is not set.");
  }
  return raw.replace(/\/+$/, "");
}

// M14.17: the TTL ceiling on every cached public read. It's the fallback safety net for M14.15's
// publish-on-read scheduling in particular — a SCHEDULED document has no explicit "now visible"
// trigger event (D-PublishOnRead: correctness lives in a WHERE clause, not a scheduler), so this
// bounds how stale a cached page can be once its publish_at passes. Every other case (an explicit
// publish/unlist/revert) is invalidated near-instantly instead, via revalidateTagRemote below.
const CACHE_REVALIDATE_SECONDS = 60;

// Tags a given fetch so a later revalidateTag(...) call (POST /api/revalidate, called by
// openfaithmap-admin right after a transition) can bust it on demand, capped by
// CACHE_REVALIDATE_SECONDS regardless. Site-level granularity for anything that can't know a
// document id ahead of the fetch itself (a path/slug lookup); document-level where it can
// (getPublicBlocks, which already takes documentId as an argument).
function client(cache?: { tags?: string[]; revalidate?: number } | "no-store") {
  if (cache === undefined) return createOpenFaithMapClient({ baseUrl: requireBaseUrl() });
  const fetchImpl: FetchFunction =
    cache === "no-store"
      ? (url, init) => fetch(url, { ...init, cache: "no-store" })
      : (url, init) => fetch(url, { ...init, next: { tags: cache.tags, revalidate: cache.revalidate ?? CACHE_REVALIDATE_SECONDS } });
  return createOpenFaithMapClient({ baseUrl: requireBaseUrl(), fetch: fetchImpl });
}

function siteTag(siteId: string): string {
  return `content-site:${siteId}`;
}

function documentTag(documentId: string): string {
  return `content-document:${documentId}`;
}

async function unwrap<T>(promise: Promise<T>): Promise<T> {
  try {
    return await promise;
  } catch (e) {
    if (isConjureError(e) && e.body && typeof e.body === "object") {
      const body = e.body as { errorName?: string; parameters?: Record<string, unknown> };
      throw new ContentApiError(e.status ?? 0, body.errorName ?? "Unknown", body.parameters ?? {});
    }
    throw e;
  }
}

export async function getSite(congregationUnitId: string): Promise<Site> {
  return unwrap(client().contentPublic.getSite(congregationUnitId));
}

// M14.9: what the tenant-subdomain route resolves a Host header's slug through. M14.10 wraps this
// in React's cache() — the new [slug]/layout.tsx and every route nested under it (page.tsx,
// preview/page.tsx, [...pageSlug]/page.tsx) each resolve the site independently, and cache()
// dedupes those to one network call per request instead of prop-drilling site data through Next's
// layout/page boundary (which doesn't support that natively).
export const getSiteBySlug = cache(async (slug: string): Promise<Site> => {
  // No tag: a site's slug/identity is effectively immutable once created — the TTL alone is enough.
  return unwrap(client({ revalidate: CACHE_REVALIDATE_SECONDS }).contentPublic.getSiteBySlug(slug));
});

// M14.11: the tenant layout's one call for header/footer data — logoUrl/socialLinks from
// content_sites, congregationName/address/schedules composed live from religion at read time.
export const getSiteChrome = cache(async (siteId: string): Promise<SiteChrome> => {
  return unwrap(client({ tags: [siteTag(siteId)] }).contentPublic.getSiteChrome(siteId));
});

export async function listPublicDocuments(siteId: string, kind?: string): Promise<Document[]> {
  const page = await unwrap(client({ tags: [siteTag(siteId)] }).contentPublic.listPublicDocuments(siteId, kind));
  return page.documents;
}

// M14.17: wrapped in cache() like getSiteBySlug/getSiteChrome/getPublicDocumentByPath — the page
// route's generateMetadata (title/description/OG derived from these same blocks) and its default
// export both resolve one document's blocks, deduped to one network call per request.
export const getPublicBlocks = cache(async (documentId: string): Promise<Block[]> => {
  const list = await unwrap(client({ tags: [documentTag(documentId)] }).contentPublic.getPublicBlocks(documentId));
  return list.blocks;
});

// M14.7: the one deliberate exception to "published/unlisted only" — gated by a site-scoped preview
// token (minted by openfaithmap-admin via ContentService.createPreviewLink) instead of a session,
// since this app never holds one. Throws ContentApiError with errorName "Content:PreviewTokenInvalid"
// for a missing/malformed/expired/wrong-site token. Never cached (M14.17): a preview's whole point
// is to reflect the draft as it stands right now (M14.7), not up to 60s ago.
export async function listPreviewDocuments(siteId: string, token: string, kind?: string): Promise<Document[]> {
  const page = await unwrap(client("no-store").contentPublic.listPreviewDocuments(siteId, token, kind));
  return page.documents;
}

export async function getPreviewBlocks(documentId: string, token: string): Promise<Block[]> {
  const list = await unwrap(client("no-store").contentPublic.getPreviewBlocks(documentId, token));
  return list.blocks;
}

// M14.10: the site's hand-built nav menu, targets already resolved to ready-to-render hrefs.
export async function listPublicNavItems(siteId: string): Promise<PublicNavItem[]> {
  const list = await unwrap(client({ tags: [siteTag(siteId)] }).contentPublic.listPublicNavItems(siteId));
  return list.items;
}

// M14.10: resolves the leaf PAGE document plus its real ancestor chain for the tenant-subdomain
// catch-all page route — path is an ordered array of URL slug segments (1-3 long), joined here
// into the slash-separated string the API expects. Throws ContentApiError with errorName
// "Content:DocumentNotFound" for any resolution failure (missing, non-PAGE, draft, or a path that
// doesn't match the document's real ancestor chain).
//
// M14.14: contentLocale is the document's own free-text content locale (content_documents.locale),
// decoupled from the site chrome's UI language (next-intl's `[locale]` route segment, fixed to 4
// values) — see the `[contentLocale]` URL segment this app's page route reads it from. Wrapped in
// cache() like getSiteBySlug/getSiteChrome: the page route's generateMetadata (hreflang) and its
// default export both resolve the same document, deduped to one network call per request.
export const getPublicDocumentByPath = cache(
  async (siteId: string, contentLocale: string, path: string[]): Promise<DocumentWithAncestors> => {
    return unwrap(client({ tags: [siteTag(siteId)] }).contentPublic.getPublicDocumentByPath(siteId, contentLocale, path.join("/")));
  },
);

// M14.17: backs app/sitemap.ts — every effectively-PUBLISHED PAGE document's resolved href.
export async function listSitemapEntries(siteId: string): Promise<SitemapEntry[]> {
  const list = await unwrap(client({ tags: [siteTag(siteId)] }).contentPublic.listSitemapEntries(siteId));
  return list.entries;
}

// M14.16, D-InAppInbox: the third genuinely anonymous write in the codebase, after moderation's
// two (lib/moderation.ts's fileReport). Always resolves — a honeypot hit or a too-fast submission
// is handled server-side and still reports success, so this app has no way to tell a real
// submission from a silently-discarded one, by design.
export async function submitContactForm(siteId: string, input: SubmitContactFormInput): Promise<void> {
  return unwrap(client().contentPublic.submitContactForm(siteId, input));
}
