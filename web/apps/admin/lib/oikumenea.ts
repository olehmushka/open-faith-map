// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Server-only: builds a go-oikumenea SDK client bound to the current session's Google ID token
// (M1, docs/modules/core-integration.md). Never called from client-side code — the token must
// never reach the browser (docs/modules/web-admin.md's session/identity invariants).
import "server-only";
import { createOikumeneaClient, type OikumeneaClient } from "oikumenea-client";

import { auth } from "@/auth";

function requireOikumeneaBaseUrl(): string {
  const raw = process.env.OIKUMENEA_BASE_URL?.trim();
  if (!raw) {
    throw new Error("OIKUMENEA_BASE_URL is not set (see web/.env.example).");
  }
  return raw.replace(/\/+$/, "");
}

/**
 * A go-oikumenea client authenticated as the currently logged-in user (their forwarded Google ID
 * token), or unauthenticated if there is no session — go-oikumenea's own PDP decides what an
 * unauthenticated caller may do, same as every other call in this app (D-Facade: OpenFaithMap makes
 * no authorization decisions of its own).
 */
export async function oikumenea(): Promise<OikumeneaClient> {
  const session = await auth();
  return createOikumeneaClient({
    baseUrl: requireOikumeneaBaseUrl(),
    token: session?.idToken,
  });
}
