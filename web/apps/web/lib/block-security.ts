// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// D-PublicSiteCSP's render-time re-validation (M14.1) — deliberate belt-and-braces with the
// write-time allowlist in internal/content/application/blockvalidation.go: rows written before
// M14.1 landed are already unvalidated in the DB, and a future block type added through M14.13's
// catalog endpoints could reintroduce an unguarded URL field without either layer alone catching
// it. Logic is hand-duplicated rather than shared — there is no cross-language shared-code
// mechanism between the Go backend and this Next.js app for this.

const ALLOWED_URL_SCHEMES = new Set(["https:", "http:", "mailto:", "tel:"]);

/**
 * Returns the URL string if its scheme is allowlisted, else undefined. Never throws — callers
 * drop the field/block rather than crash the page on a malformed or disallowed value.
 */
export function safeUrl(raw: unknown): string | undefined {
  if (typeof raw !== "string" || raw === "") return undefined;
  try {
    const u = new URL(raw);
    return ALLOWED_URL_SCHEMES.has(u.protocol) ? raw : undefined;
  } catch {
    return undefined;
  }
}

// Embed-host allowlist for social_embed.url, keyed by its declared platform — mirrors
// socialEmbedHosts in blockvalidation.go.
const SOCIAL_EMBED_HOSTS: Record<string, string[]> = {
  facebook: ["www.facebook.com", "facebook.com", "fb.watch"],
  instagram: ["www.instagram.com", "instagram.com"],
  twitter: ["twitter.com", "x.com"],
  tiktok: ["www.tiktok.com", "tiktok.com"],
};

/** safeUrl plus a host check against the block's declared platform. */
export function safeSocialEmbedUrl(raw: unknown, platform: unknown): string | undefined {
  const url = safeUrl(raw);
  if (!url) return undefined;
  const hosts = SOCIAL_EMBED_HOSTS[String(platform)];
  if (!hosts) return undefined;
  return hosts.includes(new URL(url).host) ? url : undefined;
}

// Embed-iframe host allowlist keyed by block type, so a future vimeo_embed block type (M14.13) is
// an additive entry here, not a rewrite. YouTube-only today — no vimeo_embed block type exists in
// the seeded catalog (migrations/0002_content.sql).
const EMBED_IFRAME_HOSTS: Record<string, string[]> = {
  youtube_embed: ["www.youtube.com"],
};

/**
 * Validates an iframe embed src for the given block type. youtube_embed's videoId is not itself a
 * URL — this re-checks the *constructed* `https://www.youtube.com/embed/${videoId}` string, which
 * also guards a videoId value that tries to break out of the template literal (the resulting
 * string's own scheme/host is what's actually checked here).
 */
export function safeEmbedSrc(blockTypeCode: string, src: string): string | undefined {
  const hosts = EMBED_IFRAME_HOSTS[blockTypeCode];
  if (!hosts) return undefined;
  try {
    const u = new URL(src);
    return u.protocol === "https:" && hosts.includes(u.host) ? src : undefined;
  } catch {
    return undefined;
  }
}

// Cheap defense-in-depth on top of safeEmbedSrc: YouTube video IDs are alphanumeric plus -/_.
const YOUTUBE_VIDEO_ID_RE = /^[\w-]+$/;

export function isValidYoutubeVideoId(videoId: unknown): videoId is string {
  return typeof videoId === "string" && videoId.length > 0 && YOUTUBE_VIDEO_ID_RE.test(videoId);
}
