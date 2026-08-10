// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Unified façade for openfaithmap-api's generated TypeScript SDK, this app's own copy (M4).
//
// The per-service clients under ./generated are GENERATED from the Conjure contract
// (scripts/gen-ts-client.sh) — never hand-edited. This file is the only hand-written source.
//
// Deliberately narrower than web/apps/admin's own facade (lib/openfaithmap/index.ts): this app
// holds no session, ever (D-AdminSurface), so OpenFaithMapClientOptions has NO token field at
// all — not optional, absent — and only the two genuinely anonymous services are exposed
// (DiscoveryPublicService, ContentPublicService). There is structurally no way to forward a
// bearer token from this app, by construction, not by convention.
//
// Generated IN PLACE here, not shared with web/apps/admin's copy — no workspace, no file:
// dependency to reach across (each app's Dockerfile build context is isolated to its own
// directory).

import { DefaultHttpApiBridge, type IHttpApiBridge, type IUserAgent } from "conjure-client";

import { ContentPublicService } from "./generated/content";
import { DiscoveryPublicService } from "./generated/discovery";

export * from "./generated";

export type Supplier<T> = () => T;
export type FetchFunction = (
  url: string | Request,
  init?: RequestInit,
) => Promise<Response>;

export interface OpenFaithMapClientOptions {
  /** Scheme://host[:port] of openfaithmap-api. */
  baseUrl: string | Supplier<string>;
  /** Override the fetch implementation. */
  fetch?: FetchFunction;
  /** Identifies the caller in the server's request logs. Defaults to this SDK's name/version. */
  userAgent?: IUserAgent;
}

/** The two anonymous services this app may ever call. Returned by {@link createOpenFaithMapClient}. */
export interface OpenFaithMapClient {
  readonly discoveryPublic: DiscoveryPublicService;
  readonly contentPublic: ContentPublicService;
  /** The underlying conjure HTTP bridge, for advanced use. */
  readonly bridge: IHttpApiBridge;
}

const DEFAULT_USER_AGENT: IUserAgent = {
  productName: "openfaithmap-web-client",
  productVersion: "0.0.0",
};

/**
 * Build a unified, typed openfaithmap-api client for the anonymous public site:
 *
 *   const client = createOpenFaithMapClient({ baseUrl: "https://openfaithmap-api:3000" });
 *   const result = await client.discoveryPublic.search(lat, lng, radiusM);
 */
export function createOpenFaithMapClient(
  options: OpenFaithMapClientOptions,
): OpenFaithMapClient {
  const bridge: IHttpApiBridge = new DefaultHttpApiBridge({
    baseUrl: options.baseUrl,
    userAgent: options.userAgent ?? DEFAULT_USER_AGENT,
    fetch: options.fetch as never,
  });

  return {
    discoveryPublic: new DiscoveryPublicService(bridge),
    contentPublic: new ContentPublicService(bridge),
    bridge,
  };
}
