// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.17: on-demand cache invalidation on openfaithmap-web, called right after a document
// transition or metaTitle/metaDescription edit — the tag-based half of the hybrid caching strategy
// (lib/content.ts's own 60s TTL there is the fallback safety net for M14.15's publish-on-read
// scheduling, which has no explicit trigger event to hook a call like this into).
//
// CONTENT_REVALIDATION_URL is the internal compose-network address (mirrors OPENFAITHMAP_API_BASE_URL's
// own "reach the other service by its compose service name" convention) — never
// TENANT_APEX_HOST/buildPreviewUrl's browser-facing host:port, which this server-to-server call has
// no reason to go through.
export async function revalidateContentTags(tags: string[]): Promise<void> {
  const baseUrl = process.env.CONTENT_REVALIDATION_URL?.trim();
  const secret = process.env.CONTENT_REVALIDATION_SECRET;
  if (!baseUrl || !secret) return;

  try {
    const res = await fetch(`${baseUrl.replace(/\/+$/, "")}/api/revalidate`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "x-revalidation-secret": secret },
      body: JSON.stringify({ tags }),
    });
    if (!res.ok) {
      console.error(`revalidateContentTags: openfaithmap-web returned ${res.status}`);
    }
  } catch (e) {
    // Best-effort: a transition must never fail because the OTHER app's cache couldn't be reached
    // — the 60s TTL there is the fallback that keeps this non-fatal.
    console.error("revalidateContentTags: request failed", e);
  }
}

export function siteTag(siteId: string): string {
  return `content-site:${siteId}`;
}

export function documentTag(documentId: string): string {
  return `content-document:${documentId}`;
}
