// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

"use server";

import { search, type SearchParams } from "@/lib/discovery";
import type { DiscoverySite } from "@/lib/discovery";

/** Server action the map's filter form calls to re-run the search without a full page reload. */
export async function searchAction(params: SearchParams): Promise<DiscoverySite[]> {
  return search(params);
}
