// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.17: on-demand cache invalidation for the hybrid tags+TTL caching strategy — openfaithmap-admin
// calls this right after a document transition (publish/unlist/revert/schedule) or a metaTitle/
// metaDescription edit, so an explicit change is reflected near-instantly instead of waiting out
// lib/content.ts's 60s TTL ceiling (which stays as the fallback safety net for M14.15's
// publish-on-read scheduling, which has no such explicit trigger event to hook into).
//
// Shared-secret authenticated, not session-based: this app never holds a session (D-AdminSurface),
// and the caller here is another server (openfaithmap-admin), not a browser. `/api/*` is already
// excluded from proxy.ts's own tenant-host rewriting (its matcher's `(?!api|...)` negative
// lookahead), so this route is reachable at a fixed path regardless of Host header.
import { revalidateTag } from "next/cache";
import { NextResponse, type NextRequest } from "next/server";

export async function POST(request: NextRequest): Promise<NextResponse> {
  const secret = process.env.CONTENT_REVALIDATION_SECRET;
  if (!secret || request.headers.get("x-revalidation-secret") !== secret) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }

  let tags: unknown;
  try {
    ({ tags } = await request.json());
  } catch {
    return NextResponse.json({ error: "invalid JSON body" }, { status: 400 });
  }
  if (!Array.isArray(tags) || tags.some((t) => typeof t !== "string") || tags.length === 0) {
    return NextResponse.json({ error: "tags must be a non-empty string[]" }, { status: 400 });
  }

  for (const tag of tags as string[]) {
    // Next 16's revalidateTag requires a second cacheLife-profile argument; { expire: 0 } means
    // "treat every entry carrying this tag as immediately expired" — the closest equivalent to the
    // pre-16 single-argument revalidateTag(tag) this route is meant to provide from a Route Handler
    // (updateTag's true-immediate, read-your-own-writes semantics are Server-Action-only).
    revalidateTag(tag, { expire: 0 });
  }
  return NextResponse.json({ revalidated: true, tags });
}
