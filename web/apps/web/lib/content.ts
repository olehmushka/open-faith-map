// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// openfaithmap-api's content module public reads via the generated TypeScript SDK (M4). No
// session, ever (D-AdminSurface) — ContentPublicService never asks for a token. Mirrors
// web/apps/admin's lib/content.ts public half, minus the content.manage-gated admin functions,
// which this app can never call.
import { isConjureError } from "conjure-client";

import { createOpenFaithMapClient } from "./openfaithmap";
import type { IBlock, IBlockType, IDocument, ISite } from "./openfaithmap/generated/content";

export type Site = ISite;
export type Document = IDocument;
export type Block = IBlock;
export type BlockType = IBlockType;

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

// M14.9: what the tenant-subdomain route resolves a Host header's slug through.
export async function getSiteBySlug(slug: string): Promise<Site> {
  return unwrap(client().contentPublic.getSiteBySlug(slug));
}

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
