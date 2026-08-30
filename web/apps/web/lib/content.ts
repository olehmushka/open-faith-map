// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// openfaithmap-api's content module public reads via the generated TypeScript SDK (M4). No
// session, ever (D-AdminSurface) — ContentPublicService never asks for a token. Mirrors
// web/apps/admin's lib/content.ts public half, minus the content.manage-gated admin functions,
// which this app can never call.
import { isConjureError } from "conjure-client";
import { cache } from "react";

import { createOpenFaithMapClient } from "./openfaithmap";
import type { IBlock, IBlockType, IDocument, IDocumentTranslation, IDocumentWithAncestors, IPublicNavItem, ISite, ISiteChrome } from "./openfaithmap/generated/content";

export type Site = ISite;
export type Document = IDocument;
export type Block = IBlock;
export type BlockType = IBlockType;
export type PublicNavItem = IPublicNavItem;
export type DocumentWithAncestors = IDocumentWithAncestors;
export type DocumentTranslation = IDocumentTranslation;
export type SiteChrome = ISiteChrome;

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

function client() {
  return createOpenFaithMapClient({ baseUrl: requireBaseUrl() });
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
  return unwrap(client().contentPublic.getSiteBySlug(slug));
});

// M14.11: the tenant layout's one call for header/footer data — logoUrl/socialLinks from
// content_sites, congregationName/address/schedules composed live from religion at read time.
export const getSiteChrome = cache(async (siteId: string): Promise<SiteChrome> => {
  return unwrap(client().contentPublic.getSiteChrome(siteId));
});

export async function listPublicDocuments(siteId: string, kind?: string): Promise<Document[]> {
  const page = await unwrap(client().contentPublic.listPublicDocuments(siteId, kind));
  return page.documents;
}

export async function getPublicBlocks(documentId: string): Promise<Block[]> {
  const list = await unwrap(client().contentPublic.getPublicBlocks(documentId));
  return list.blocks;
}

// M14.7: the one deliberate exception to "published/unlisted only" — gated by a site-scoped preview
// token (minted by openfaithmap-admin via ContentService.createPreviewLink) instead of a session,
// since this app never holds one. Throws ContentApiError with errorName "Content:PreviewTokenInvalid"
// for a missing/malformed/expired/wrong-site token.
export async function listPreviewDocuments(siteId: string, token: string, kind?: string): Promise<Document[]> {
  const page = await unwrap(client().contentPublic.listPreviewDocuments(siteId, token, kind));
  return page.documents;
}

export async function getPreviewBlocks(documentId: string, token: string): Promise<Block[]> {
  const list = await unwrap(client().contentPublic.getPreviewBlocks(documentId, token));
  return list.blocks;
}

// M14.10: the site's hand-built nav menu, targets already resolved to ready-to-render hrefs.
export async function listPublicNavItems(siteId: string): Promise<PublicNavItem[]> {
  const list = await unwrap(client().contentPublic.listPublicNavItems(siteId));
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
    return unwrap(client().contentPublic.getPublicDocumentByPath(siteId, contentLocale, path.join("/")));
  },
);
