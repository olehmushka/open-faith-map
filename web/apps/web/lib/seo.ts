// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.17: derives a fallback <title>/description from a document's own blocks when no explicit
// metaTitle/metaDescription override is set (Document has no title field at all — content.md's own
// gap, restated by this milestone). Reuses the exact richText node shapes lib/rich-text.tsx already
// renders, walked here for plain text instead of React elements — never a second parser.
import { headers } from "next/headers";

import type { Block } from "@/lib/content";
import { protocolForHost } from "@/lib/tenant-host";

const DESCRIPTION_MAX_LENGTH = 160;

interface TextNode {
  type: "text";
  text?: unknown;
}

interface ListItemNode {
  content?: unknown;
}

interface ListNode {
  type: "list";
  items?: unknown;
}

function richTextToPlainText(nodes: unknown): string {
  if (!Array.isArray(nodes)) return "";
  return nodes
    .map((node) => {
      if (!node || typeof node !== "object") return "";
      const n = node as TextNode | ListNode;
      if (n.type === "text") return String((n as TextNode).text ?? "");
      if (n.type === "list") {
        const items = Array.isArray((n as ListNode).items) ? ((n as ListNode).items as ListItemNode[]) : [];
        return items.map((item) => richTextToPlainText(item?.content)).join(" ");
      }
      return "";
    })
    .join("")
    .replace(/\s+/g, " ")
    .trim();
}

/** kebab-case slug -> "Title Case". Shared with components/breadcrumbs.tsx's own label derivation
 * — a document has no title field, so both the breadcrumb label and this fallback title need the
 * same "humanize the slug" last resort. */
export function humanizeSlug(slug: string): string {
  return slug
    .split("-")
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

/** First heading block's plain text, else the humanized slug — never empty. */
export function deriveTitle(blocks: Block[], slug: string): string {
  const sorted = [...blocks].sort((a, b) => a.position - b.position);
  for (const b of sorted) {
    if (b.blockTypeCode !== "heading") continue;
    const data = (b.data ?? {}) as Record<string, unknown>;
    const text = richTextToPlainText(data.text);
    if (text) return text;
  }
  return humanizeSlug(slug);
}

// Block types whose most prominent text-bearing field is worth summarizing, in the field each
// carries it under — checked in document order, not this list's order. Deliberately excludes
// "heading": deriveTitle already surfaces the first heading as the <title>, and a description
// that's just a copy of the title (the common case, since a heading is usually a document's first
// block) tells a search result reader nothing extra.
const DESCRIPTION_FIELDS: Record<string, string> = {
  paragraph: "text",
  quote: "text",
  list: "content",
};

/** First non-empty text-bearing block's plain text, truncated — undefined if the document has no
 * such block (an image-only or contact-form-only page, say). */
export function deriveDescription(blocks: Block[]): string | undefined {
  const sorted = [...blocks].sort((a, b) => a.position - b.position);
  for (const b of sorted) {
    const field = DESCRIPTION_FIELDS[b.blockTypeCode];
    if (!field) continue;
    const data = (b.data ?? {}) as Record<string, unknown>;
    const text = richTextToPlainText(data[field]);
    if (text) return truncate(text, DESCRIPTION_MAX_LENGTH);
  }
  return undefined;
}

function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength - 1).trimEnd() + "…";
}

// M14.14 established this exact host -> origin heuristic in the page route's own generateMetadata
// (hreflang); shared here so canonical/OG/JSON-LD URLs use the identical rule instead of a second
// copy drifting from it.
export async function resolveOrigin(): Promise<string | null> {
  const headersList = await headers();
  // proxy.ts's own next-intl-response-header-copy can duplicate "host" onto the rewritten request
  // in some cases — take the first value defensively rather than emitting a broken two-host URL.
  const host = headersList.get("host")?.split(",")[0]?.trim();
  if (!host) return null;
  return `${protocolForHost(host)}://${host}`;
}

/** Escapes '<' as its < JSON escape — required before rendering JSON-LD as a `<script>`
 * child: a literal "</script>" inside admin-controlled text (a congregation name, an event title)
 * would otherwise close the tag early in the browser's HTML parser, independent of React's own
 * text-node handling (which never HTML-escapes plain text children). Standard mitigation for this
 * exact pattern. */
export function jsonLdScript(data: unknown): string {
  return JSON.stringify(data).replace(/</g, "\\u003c");
}
