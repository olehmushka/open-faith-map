// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { DiscoveryMap } from "./discovery-map-loader";
import { search } from "@/lib/discovery";

// Always rendered per-request, never statically prerendered — the discovery cache changes
// independently of any build, and this app has no signal (no auth(), no cookies()) that would
// otherwise tell Next.js to skip static generation. Without this, `next build` tries to reach
// openfaithmap-api at build time, before OPENFAITHMAP_API_BASE_URL is even set.
export const dynamic = "force-dynamic";

// M4: the public discovery map/search (docs/modules/discovery.md). Server-fetches an unfiltered
// initial result set (served from discovery_site_cache, or a live go-oikumenea call via the
// service principal on a cache miss — never this app calling go-oikumenea directly); the map's
// filter form re-runs the search client-side via a server action.
export default async function Home() {
  const initialSites = await search({});

  return (
    <main className="flex min-h-screen flex-col">
      <header className="mx-auto w-full max-w-5xl px-6 pt-8">
        <h1 className="text-2xl font-semibold">OpenFaithMap</h1>
        <p className="text-sm text-gray-600">Find a church near you.</p>
      </header>
      <div className="mx-auto w-full max-w-5xl flex-1">
        <DiscoveryMap initialSites={initialSites} />
      </div>
    </main>
  );
}
