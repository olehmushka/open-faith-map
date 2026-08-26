// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { getTranslations } from "next-intl/server";

import { DiscoveryShell } from "../discovery/discovery-shell-loader";
import { facets, search } from "@/lib/discovery";
import { parseFilters, searchParamsFromRecord } from "@/lib/discovery-url-state";
import { LocaleSwitcher } from "./locale-switcher";

// Always rendered per-request, never statically prerendered — the discovery cache changes
// independently of any build, and this app has no signal (no auth(), no cookies()) that would
// otherwise tell Next.js to skip static generation. Without this, `next build` tries to reach
// openfaithmap-api at build time, before OPENFAITHMAP_API_BASE_URL is even set. Also required
// now that the layout's generateStaticParams() declares a 4-locale param space — this override
// keeps every locale variant of this page dynamic too.
export const dynamic = "force-dynamic";

// M4: the public discovery map/search (docs/modules/discovery.md). Server-fetches an unfiltered
// initial result set (served from discovery_site_cache, or a live go-oikumenea call via the
// service principal on a cache miss — never this app calling go-oikumenea directly); the map's
// filter form re-runs the search client-side via a server action.
export default async function Home({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const t = await getTranslations("HomePage");
  const filters = parseFilters(searchParamsFromRecord(await searchParams));
  const [initialSites, initialFacets] = await Promise.all([
    search({
      tradition: filters.tradition,
      language: filters.language,
      lat: filters.lat,
      lng: filters.lng,
      radiusM: filters.radiusM,
    }),
    facets(),
  ]);

  return (
    <main className="flex min-h-screen flex-col">
      <header className="flex w-full items-start justify-between px-6 pt-6 pb-2">
        <div>
          <h1 className="text-2xl font-semibold">{t("title")}</h1>
          <p className="text-sm text-gray-600">{t("tagline")}</p>
        </div>
        <LocaleSwitcher />
      </header>
      <div className="w-full flex-1">
        <DiscoveryShell initialSites={initialSites} facets={initialFacets} />
      </div>
    </main>
  );
}
