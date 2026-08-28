// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { NextResponse, type NextRequest } from "next/server";

import { getSite } from "@/lib/content";

// M14.9/D-TenantSubdomains: this used to be the public one-pager (keyed by the go-oikumenea
// congregation unit RID); rendering now lives on the tenant subdomain (components/site-page.tsx,
// reached via app/[locale]/%5Fsites/[slug]/page.tsx). This route survives only to 301 any existing
// or indexed link to its new home — a real 301, which requires a Route Handler: Next's own
// redirect()/permanentRedirect() send 307/308, not 301.
//
// Deliberately redirects to the tenant root, not the caller's original locale/path — next-intl's
// own proxy.ts redirect re-adds the locale prefix on the next hop. Simpler, and correct in every
// case since this route was never itself locale-aware beyond the prefix.
export async function GET(request: NextRequest, { params }: { params: Promise<{ unitId: string }> }) {
  const { unitId } = await params;
  const site = await getSite(unitId).catch(() => null);
  if (!site) {
    return new NextResponse(null, { status: 404 });
  }

  const host = request.headers.get("host") ?? request.nextUrl.host;
  const hostname = host.split(":")[0];
  const port = host.includes(":") ? host.split(":")[1] : "";

  const target = new URL(request.url);
  target.hostname = `${site.slug}.${hostname}`;
  target.port = port;
  target.pathname = "/";
  target.search = "";

  return NextResponse.redirect(target, 301);
}
