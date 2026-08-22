// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Unified façade for openfaithmap-api's generated TypeScript SDK (M2.6, D-Stack).
//
// The per-service clients under ./generated are GENERATED from the Conjure contract
// (scripts/gen-ts-client.sh) — never hand-edited. This file is the only hand-written source: it
// wires the generated service onto one shared HTTP bridge (one base URL + one bearer token) — the
// same pattern go-oikumenea's own TypeScript SDK used to use before M10.7's lib/oikumenea.ts
// (deleted this milestone) was retired along with go-oikumenea itself.
//
// Deliberately generated IN PLACE here, not as a separate published package like go-oikumenea's
// oikumenea-client — web/apps/admin is this SDK's only consumer, in this repo, and its Dockerfile's
// build context is isolated to its own directory (no workspace, nothing to reach across).

import { DefaultHttpApiBridge, type IHttpApiBridge, type IUserAgent } from "conjure-client";

import { CongregationImportService } from "./generated/congregationimport";
import { ContentPublicService, ContentService } from "./generated/content";
import { CorePublicService, CoreService, CoreSuperAdminService } from "./generated/core";
import { DiscoveryPublicService, DiscoveryService } from "./generated/discovery";
import { ModerationPublicService, ModerationService } from "./generated/moderation";
import { RegistrationService } from "./generated/registration";
import { VouchingService } from "./generated/vouching";

export * from "./generated";

/** A value or a (possibly async) supplier of it — matches conjure-client's token/baseUrl inputs. */
export type Supplier<T> = () => T;
export type FetchFunction = (
  url: string | Request,
  init?: RequestInit,
) => Promise<Response>;

export interface OpenFaithMapClientOptions {
  /** Scheme://host[:port] of openfaithmap-api. */
  baseUrl: string | Supplier<string>;
  /** Bearer token (the caller's forwarded Google ID token). May be omitted for an unauthenticated call. */
  token?: string | Supplier<string> | Supplier<Promise<string>>;
  /** Override the fetch implementation. */
  fetch?: FetchFunction;
  /** Identifies the caller in the server's request logs. Defaults to this SDK's name/version. */
  userAgent?: IUserAgent;
}

/** Every generated service, bound to one shared HTTP bridge. Returned by {@link createOpenFaithMapClient}. */
export interface OpenFaithMapClient {
  readonly registration: RegistrationService;
  readonly content: ContentService;
  readonly contentPublic: ContentPublicService;
  readonly discovery: DiscoveryService;
  readonly discoveryPublic: DiscoveryPublicService;
  readonly moderation: ModerationService;
  readonly moderationPublic: ModerationPublicService;
  readonly vouching: VouchingService;
  readonly congregationImport: CongregationImportService;
  readonly core: CoreService;
  readonly coreSuperAdmin: CoreSuperAdminService;
  readonly corePublic: CorePublicService;
  /** The underlying conjure HTTP bridge, for advanced use. */
  readonly bridge: IHttpApiBridge;
}

const DEFAULT_USER_AGENT: IUserAgent = {
  productName: "openfaithmap-client",
  productVersion: "0.0.0",
};

/**
 * Build a unified, typed openfaithmap-api client. One transport config (base URL + bearer) powers
 * every service:
 *
 *   const client = createOpenFaithMapClient({ baseUrl: "https://localhost:3000", token });
 *   const page = await client.registration.listRequests();
 */
export function createOpenFaithMapClient(
  options: OpenFaithMapClientOptions,
): OpenFaithMapClient {
  const bridge: IHttpApiBridge = new DefaultHttpApiBridge({
    baseUrl: options.baseUrl,
    userAgent: options.userAgent ?? DEFAULT_USER_AGENT,
    token: options.token,
    fetch: options.fetch as never,
  });

  return {
    registration: new RegistrationService(bridge),
    content: new ContentService(bridge),
    contentPublic: new ContentPublicService(bridge),
    discovery: new DiscoveryService(bridge),
    discoveryPublic: new DiscoveryPublicService(bridge),
    moderation: new ModerationService(bridge),
    moderationPublic: new ModerationPublicService(bridge),
    vouching: new VouchingService(bridge),
    congregationImport: new CongregationImportService(bridge),
    core: new CoreService(bridge),
    coreSuperAdmin: new CoreSuperAdminService(bridge),
    corePublic: new CorePublicService(bridge),
    bridge,
  };
}
