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
  ICreateDocumentRequest,
  ICreateSiteRequest,
  IDocument,
  ISite,
  IUpdateDocumentRequest,
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

export async function transitionDocument(documentId: string, action: DocumentTransitionAction): Promise<Document> {
  return unwrap((await client()).content.transitionDocument(documentId, { action }));
}

export async function getBlocks(documentId: string): Promise<Block[]> {
  const list = await unwrap((await client()).content.getBlocks(documentId));
  return list.blocks;
}

export async function putBlocks(documentId: string, blocks: BlockInput[]): Promise<Block[]> {
  const list = await unwrap((await client()).content.putBlocks(documentId, { blocks }));
  return list.blocks;
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
