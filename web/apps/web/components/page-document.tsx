// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { Blocks } from "@/app/blocks";
import { getPublicBlocks } from "@/lib/content";

// M14.10: the single-Page renderer, mirroring components/site-page.tsx's own EventBlocks/PostBlocks
// sub-pattern — fetch one document's blocks, render them. Used by the tenant-subdomain catch-all
// page route ([...pageSlug]/page.tsx), which already has the resolved documentId in hand (via
// getPublicDocumentByPath) and needs no further fetch of the document itself.
export async function PageDocument({ documentId, siteId }: { documentId: string; siteId: string }) {
  const blocks = await getPublicBlocks(documentId);
  return <Blocks blocks={blocks} siteId={siteId} />;
}
