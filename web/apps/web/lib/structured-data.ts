// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.17: builds the plain-object JSON-LD payloads components/json-ld.tsx renders — kept separate
// from the components that call them so the schema.org shapes are easy to scan/extend without
// wading through render logic. No external JSON-LD library: three fixed shapes, not worth a
// dependency.
import type { Document, SiteChrome } from "@/lib/content";
import { humanizeSlug } from "@/lib/seo";

export function churchJsonLd(chrome: SiteChrome, url: string) {
  const sameAs = [
    chrome.socialLinks.facebook,
    chrome.socialLinks.instagram,
    chrome.socialLinks.youtube,
    chrome.socialLinks.twitter,
    chrome.socialLinks.website,
  ].filter((v): v is string => Boolean(v));

  return {
    "@context": "https://schema.org",
    "@type": "Church",
    name: chrome.congregationName,
    url,
    ...(chrome.address ? { address: chrome.address } : {}),
    ...(chrome.logoUrl ? { logo: chrome.logoUrl, image: chrome.logoUrl } : {}),
    ...(sameAs.length > 0 ? { sameAs } : {}),
    // No telephone: religion.Site (this data's own source, GetSiteChrome) carries no phone/email
    // field at all — a named gap, not an oversight.
    ...(chrome.latitude != null && chrome.longitude != null
      ? { geo: { "@type": "GeoCoordinates", latitude: chrome.latitude, longitude: chrome.longitude } }
      : {}),
  };
}

export function eventJsonLd(name: string, startDate: string, endDate: string | undefined, url: string, address: string | undefined) {
  return {
    "@context": "https://schema.org",
    "@type": "Event",
    name,
    startDate,
    ...(endDate ? { endDate } : {}),
    url,
    // location = the site's own address: an EVENT document has no venue field of its own
    // (content_documents carries no location column), a named scope call rather than an
    // oversight — see M14.17's own docs.
    ...(address ? { location: { "@type": "Place", name, address } } : {}),
  };
}

/** chain is root-first, leaf last (ancestors + the current document, in that order) — matches
 * components/breadcrumbs.tsx's own `[...ancestors, current]` convention. hrefFor receives the
 * slug chain up to and including position i (1-indexed length), matching Breadcrumbs' own
 * per-item href derivation. */
export function breadcrumbListJsonLd(chain: Document[], hrefFor: (slugChain: string[]) => string) {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: chain.map((doc, i) => ({
      "@type": "ListItem",
      position: i + 1,
      name: humanizeSlug(doc.slug),
      item: hrefFor(chain.slice(0, i + 1).map((d) => d.slug)),
    })),
  };
}
