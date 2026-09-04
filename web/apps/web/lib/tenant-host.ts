// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Pure Host-header parsing for the tenant-subdomain proxy (M14.9, D-TenantSubdomains). Kept
// separate from proxy.ts and unit-tested in isolation (mirrors block-security.ts's own precedent
// of keeping security-sensitive parsing logic in a plain, vitest-covered module) since a mistake
// here is either "the apex host stops serving discovery" or "a tenant site leaks onto the apex" —
// not something to get right only by eyeballing proxy.ts.

const SITES_PATH_RE = /^\/(?:[a-z]{2}\/)?_sites(?:\/|$)/;

// APEX_HOST defaults to "localhost" so `*.localhost` verifies with zero config (browsers resolve
// it to loopback, no DNS needed) — forward-compatible with M14.18's real apex domain, which will
// set this env var without any code change here.
export function parseApexHost(env?: string): string {
  return (env ?? process.env.APEX_HOST ?? "localhost").trim().toLowerCase();
}

// resolveTenantSlug returns the congregation slug a request's Host header names, or null if the
// host is the bare apex, `www.<apex>`, or not a recognized subdomain of the apex at all (an
// unrecognized host is never guessed at — treated the same as the apex).
export function resolveTenantSlug(hostname: string, apexHost: string): string | null {
  const host = hostname.split(":")[0]?.toLowerCase() ?? "";
  if (host === apexHost || host === `www.${apexHost}`) {
    return null;
  }
  const suffix = `.${apexHost}`;
  if (!host.endsWith(suffix)) {
    return null;
  }
  const labelPart = host.slice(0, -suffix.length);
  if (!labelPart || labelPart.includes(".")) {
    // Nested subdomains (foo.bar.<apex>) aren't a tenant shape D-TenantSubdomains defines — treat
    // as unrecognized rather than guessing which label is the slug.
    return null;
  }
  return labelPart;
}

// isSitesPath guards the apex-host "/_sites/* must 404" boundary (D-TenantSubdomains) — checked
// against both the raw pathname (a client hitting /_sites/grace directly, no locale segment yet)
// and an already locale-prefixed one (/en/_sites/grace).
export function isSitesPath(pathname: string): boolean {
  return SITES_PATH_RE.test(pathname);
}

// M14.17: local dev's Host header is never the bare "localhost" for a tenant request — it's
// "{slug}.localhost[:port]" (M14.9's own "*.localhost resolves to loopback with no DNS" design).
// A naive `host.startsWith("localhost")` (M14.14's original hreflang code, now centralized here)
// therefore misclassifies every tenant subdomain as https in local dev — checking the hostname's
// own suffix, not its prefix, is what a subdomain actually needs.
export function protocolForHost(hostname: string): "http" | "https" {
  const host = hostname.split(":")[0]?.toLowerCase() ?? "";
  if (host === "localhost" || host.endsWith(".localhost") || host.startsWith("127.")) {
    return "http";
  }
  return "https";
}

// injectSitesSegment rewrites a locale-prefixed pathname into the internal tenant tree, e.g.
// "/en/about" + "grace" -> "/en/_sites/grace/about". Assumes the locale prefix is already present
// (proxy.ts only calls this after next-intl's own middleware has resolved it) — never called with
// a bare, non-locale-prefixed pathname.
export function injectSitesSegment(pathname: string, slug: string): string {
  const match = pathname.match(/^\/([a-z]{2})(\/.*)?$/);
  if (!match) {
    return `/_sites/${slug}${pathname}`;
  }
  const [, locale, rest] = match;
  return `/${locale}/_sites/${slug}${rest ?? ""}`;
}
