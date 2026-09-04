// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.17: renders one JSON-LD block as {JSON.stringify(...)} text content — never
// dangerouslySetInnerHTML (M14.1's ESLint invariant: no `dangerouslySetInnerHTML` anywhere in this
// app). jsonLdScript's own `<` escaping is what keeps this safe against admin-controlled text (a
// congregation name, an event title) containing a literal "</script>" — see its own comment.
import { jsonLdScript } from "@/lib/seo";

export function JsonLd({ data }: { data: unknown }) {
  return <script type="application/ld+json">{jsonLdScript(data)}</script>;
}
