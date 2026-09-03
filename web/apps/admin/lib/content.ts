// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: openfaithmap-api's content module via the generated TypeScript SDK
// (./openfaithmap, M3), same shape as lib/registration.ts (M2.6). Forwards the session's Google ID
// token unchanged, same as lib/core.ts. Public reads (getSite/listPublicDocuments/
// getPublicBlocks/listBlockTypes) go through the SAME client — ContentPublicService's generated
// methods simply never look at the bearer token — so no separate unauthenticated client is needed.
import "server-only";

import { isConjureError } from "conjure-client";

import { auth } from "@/auth";

import { createOpenFaithMapClient } from "./openfaithmap";
import type {
  IBlock,
  IBlockInput,
  IBlockType,
  IContactFormSubmission,
  ICreateBlockTypeRequest,
  ICreateDocumentRequest,
  ICreatePatternRequest,
  ICreateSiteRequest,
  IDocument,
  IDocumentRevision,
  INavItem,
  INavItemInput,
  IPattern,
  ISite,
  ISocialLinks,
  IUpdateBlockTypeRequest,
  IUpdateDocumentRequest,
  IUpdatePatternRequest,
} from "./openfaithmap/generated/content";
import { DocumentTransitionAction } from "./openfaithmap/generated/content";

export type Site = ISite;
export type Document = IDocument;
export type Block = IBlock;
export type BlockType = IBlockType;
export type CreateSiteInput = ICreateSiteRequest;
export type CreateDocumentInput = ICreateDocumentRequest;
export type UpdateDocumentInput = IUpdateDocumentRequest;
export type BlockInput = IBlockInput;
export type DocumentRevision = IDocumentRevision;
export type NavItem = INavItem;
export type NavItemInput = INavItemInput;
export type SocialLinks = ISocialLinks;
export type Pattern = IPattern;
export type CreateBlockTypeInput = ICreateBlockTypeRequest;
export type UpdateBlockTypeInput = IUpdateBlockTypeRequest;
export type CreatePatternInput = ICreatePatternRequest;
export type UpdatePatternInput = IUpdatePatternRequest;
export type ContactFormSubmission = IContactFormSubmission;
export { DocumentTransitionAction };

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

async function client() {
  const session = await auth();
  return createOpenFaithMapClient({
    baseUrl: requireBaseUrl(),
    token: session?.idToken,
    // M11.3, D-SessionTracking: every authenticated request needs a valid X-Session-Id alongside
    // the bearer (internal/identity/middleware.Authenticator.Handle) — omitted here until now,
    // silently 401-ing every call through this file since M11.3 shipped (see lib/core.ts's own
    // client() for the pattern this mirrors).
    fetch: session?.sessionId
      ? (url, init) =>
          fetch(url, { ...init, headers: { ...init?.headers, "X-Session-Id": session.sessionId! } })
      : undefined,
  });
}

/** Translates a ConjureError (the SDK's transport-level error) into the errorName/parameters shape callers already handle. */
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

// ---- admin (content.manage-gated) ----

export async function createSite(input: CreateSiteInput): Promise<Site> {
  return unwrap((await client()).content.createSite(input));
}

export async function updateSiteTheme(siteId: string, theme: unknown): Promise<Site> {
  return unwrap((await client()).content.updateSiteTheme(siteId, { theme }));
}

// M14.11: logoUrl/socialLinks are content_sites' own settings, full-replace like updateSiteTheme.
export async function updateSiteChrome(siteId: string, logoUrl: string | null, socialLinks: SocialLinks): Promise<Site> {
  return unwrap((await client()).content.updateSiteChrome(siteId, { logoUrl, socialLinks }));
}

export async function listDocuments(siteId: string): Promise<Document[]> {
  const page = await unwrap((await client()).content.listDocuments(siteId));
  return page.documents;
}

export async function createDocument(siteId: string, input: CreateDocumentInput): Promise<Document> {
  return unwrap((await client()).content.createDocument(siteId, input));
}

export async function updateDocument(documentId: string, input: UpdateDocumentInput): Promise<Document> {
  return unwrap((await client()).content.updateDocument(documentId, input));
}

// publishAt is only meaningful for DocumentTransitionAction.SCHEDULE (M14.15) — required (and must
// be in the future) there, ignored for every other action.
export async function transitionDocument(documentId: string, action: DocumentTransitionAction, publishAt?: string): Promise<Document> {
  return unwrap((await client()).content.transitionDocument(documentId, { action, publishAt: publishAt ?? null }));
}

export async function getBlocks(documentId: string): Promise<Block[]> {
  const list = await unwrap((await client()).content.getBlocks(documentId));
  return list.blocks;
}

export async function putBlocks(documentId: string, blocks: BlockInput[]): Promise<Block[]> {
  const list = await unwrap((await client()).content.putBlocks(documentId, { blocks }));
  return list.blocks;
}

// M14.6: history list, newest first, excluding the in-progress draft — and restoring a past
// checkpoint into the draft (never auto-publishing; Publish is still a separate, explicit step).
export async function listRevisions(documentId: string): Promise<DocumentRevision[]> {
  const page = await unwrap((await client()).content.listRevisions(documentId));
  return page.revisions;
}

export async function restoreRevision(documentId: string, revisionId: string): Promise<Block[]> {
  const list = await unwrap((await client()).content.restoreRevision(documentId, revisionId));
  return list.blocks;
}

// M14.7: mints a short-lived, site-scoped token for openfaithmap-web's preview route to accept
// instead of a session (that app never holds one). Handed to the visitor as a URL query param, never
// used again here.
export async function createPreviewLink(siteId: string): Promise<string> {
  const link = await unwrap((await client()).content.createPreviewLink(siteId));
  return link.token;
}

// M14.10: the site's hand-built nav menu — putNavItems is a full replace (same shape as putBlocks
// before M14.6's revision refactor), so the editor always saves the whole ordered list as one call.
export async function listNavItems(siteId: string): Promise<NavItem[]> {
  const list = await unwrap((await client()).content.listNavItems(siteId));
  return list.items;
}

export async function putNavItems(siteId: string, items: NavItemInput[]): Promise<NavItem[]> {
  const list = await unwrap((await client()).content.putNavItems(siteId, { items }));
  return list.items;
}

// ---- block-type/pattern catalog admin (M14.13, content.catalog.manage — platform-moderator, NOT
// content.manage; a non-moderator's call comes back Content:Forbidden, same discipline
// app/[locale]/admin/moderation/page.tsx already follows for moderation.standing) ----

export async function listBlockTypesForCatalog(): Promise<BlockType[]> {
  const page = await unwrap((await client()).content.listBlockTypesForCatalog());
  return page.blockTypes;
}

export async function createBlockType(input: CreateBlockTypeInput): Promise<BlockType> {
  return unwrap((await client()).content.createBlockType(input));
}

export async function updateBlockType(blockTypeId: string, input: UpdateBlockTypeInput): Promise<BlockType> {
  return unwrap((await client()).content.updateBlockType(blockTypeId, input));
}

export async function createPattern(input: CreatePatternInput): Promise<Pattern> {
  return unwrap((await client()).content.createPattern(input));
}

export async function updatePattern(patternId: string, input: UpdatePatternInput): Promise<Pattern> {
  return unwrap((await client()).content.updatePattern(patternId, input));
}

export async function deletePattern(patternId: string): Promise<void> {
  return unwrap((await client()).content.deletePattern(patternId));
}

// listFormSubmissions backs the Messages screen (M14.16, D-InAppInbox) — content.manage-gated,
// server-side, same as every other content.manage call in this file (no local role check here).
export async function listFormSubmissions(siteId: string): Promise<ContactFormSubmission[]> {
  const page = await unwrap((await client()).content.listFormSubmissions(siteId));
  return page.submissions;
}

// buildPreviewUrl points at the SAME tenant host every other tenant page uses (D-TenantSubdomains) —
// "{slug}.{apex}", never a path under this admin app's own origin, so the WordPress-CVE-style
// cross-origin guarantee holds by construction. TENANT_APEX_HOST is openfaithmap-web's own published
// host:port in local dev ("localhost:3002") and the real apex once U14 resolves; this app has no
// other reason to know that host today, so it isn't folded into OPENFAITHMAP_API_BASE_URL.
export function buildPreviewUrl(site: Site, locale: string, token: string): string {
  const apexHost = process.env.TENANT_APEX_HOST?.trim();
  if (!apexHost) {
    throw new Error("TENANT_APEX_HOST is not set.");
  }
  const protocol = apexHost.startsWith("localhost") ? "http" : "https";
  return `${protocol}://${site.slug}.${apexHost}/${locale}/preview?token=${encodeURIComponent(token)}`;
}

// ---- public (no content.manage required — published/unlisted only) ----

export async function getSite(congregationUnitId: string): Promise<Site> {
  return unwrap((await client()).contentPublic.getSite(congregationUnitId));
}

export async function listPublicDocuments(siteId: string): Promise<Document[]> {
  const page = await unwrap((await client()).contentPublic.listPublicDocuments(siteId));
  return page.documents;
}

export async function getPublicBlocks(documentId: string): Promise<Block[]> {
  const list = await unwrap((await client()).contentPublic.getPublicBlocks(documentId));
  return list.blocks;
}

export async function listBlockTypes(): Promise<BlockType[]> {
  const page = await unwrap((await client()).contentPublic.listBlockTypes());
  return page.blockTypes;
}

// M14.13: not sensitive data, same reasoning listBlockTypes above already uses for having no auth
// requirement of its own. The document editor's insert-a-pattern UI calls this to fetch a
// pattern's blocks to copy client-side (unsynced — see Pattern's own doc comment).
export async function listPatterns(): Promise<Pattern[]> {
  const page = await unwrap((await client()).contentPublic.listPatterns());
  return page.patterns;
}
