import { NextResponse, type NextRequest } from "next/server";
import createMiddleware from "next-intl/middleware";

import { routing } from "./i18n/routing";
import { injectSitesSegment, isSitesPath, parseApexHost, resolveTenantSlug } from "./lib/tenant-host";

const intlMiddleware = createMiddleware(routing);

// Host-header tenant resolution (M14.9, D-TenantSubdomains), composed with next-intl's own
// locale middleware rather than a separate middleware.ts — Next 16 renamed that entrypoint to
// proxy.ts, and there is exactly one of these per app.
export default function proxy(request: NextRequest) {
  const apexHost = parseApexHost();
  const hostname = request.headers.get("host") ?? request.nextUrl.hostname;
  const slug = resolveTenantSlug(hostname, apexHost);
  const pathname = request.nextUrl.pathname;

  // Guardrail first, before anything else runs: direct /_sites/* access from the apex host must
  // 404 — the boundary that keeps Phase 1's single-app shape from collapsing into one
  // undifferentiated route tree. Checked before next-intl even sees the request.
  if (slug === null && isSitesPath(pathname)) {
    return new NextResponse(null, { status: 404 });
  }

  if (slug === null) {
    // Apex host: discovery, search, and the registration entry point keep working exactly as
    // today.
    return intlMiddleware(request);
  }

  // Tenant host. Run next-intl on the original, unmodified request first — if it wants to
  // redirect (adding the missing locale prefix, since localePrefix: "always"), let that redirect
  // go out as-is: same host, browser re-requests, and lands back here already locale-prefixed on
  // the next pass. Injecting /_sites/{slug} must be a rewrite, never visible in the address bar,
  // so it can only happen once the locale prefix is already settled.
  const intlResponse = intlMiddleware(request);
  if (intlResponse.headers.get("location")) {
    return intlResponse;
  }

  const targetUrl = request.nextUrl.clone();
  targetUrl.pathname = injectSitesSegment(pathname, slug);
  const rewritten = NextResponse.rewrite(targetUrl, { request });

  // next-intl's own response may already carry headers (notably the NEXT_LOCALE cookie via
  // Set-Cookie) even in this pass-through case — copy them onto the rewrite response we're
  // actually returning, since replacing it wholesale would silently drop them.
  intlResponse.headers.forEach((value, key) => {
    rewritten.headers.append(key, value);
  });

  return rewritten;
}

export const config = {
  matcher: ["/((?!api|_next|_vercel|.*\\..*).*)"],
};
