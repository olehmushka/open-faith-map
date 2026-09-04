// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";

import { injectSitesSegment, isSitesPath, protocolForHost, resolveTenantSlug } from "./tenant-host";

describe("resolveTenantSlug", () => {
  const apex = "localhost";

  it("returns null for the bare apex host", () => {
    expect(resolveTenantSlug("localhost", apex)).toBeNull();
    expect(resolveTenantSlug("localhost:3002", apex)).toBeNull();
  });

  it("returns null for www", () => {
    expect(resolveTenantSlug("www.localhost:3002", apex)).toBeNull();
  });

  it("resolves a tenant subdomain, with and without a port", () => {
    expect(resolveTenantSlug("grace.localhost:3002", apex)).toBe("grace");
    expect(resolveTenantSlug("grace.localhost", apex)).toBe("grace");
  });

  it("returns null for a host that doesn't end in the apex", () => {
    expect(resolveTenantSlug("grace.example.org", apex)).toBeNull();
    expect(resolveTenantSlug("evil-localhost", apex)).toBeNull();
  });

  it("returns null for a nested subdomain rather than guessing a label", () => {
    expect(resolveTenantSlug("foo.bar.localhost:3002", apex)).toBeNull();
  });

  it("does not special-case reserved-looking slugs — that's enforced server-side, not here", () => {
    expect(resolveTenantSlug("admin.localhost:3002", apex)).toBe("admin");
  });
});

describe("isSitesPath", () => {
  it.each(["/_sites/grace", "/_sites", "/_sites/grace/about", "/en/_sites/grace", "/uk/_sites/grace/about"])(
    "matches %s",
    (path) => {
      expect(isSitesPath(path)).toBe(true);
    },
  );

  it.each(["/", "/about", "/en/about", "/sites/grace", "/_sitesnot/grace"])("does not match %s", (path) => {
    expect(isSitesPath(path)).toBe(false);
  });
});

describe("protocolForHost", () => {
  it("is http for the bare apex, with or without a port", () => {
    expect(protocolForHost("localhost")).toBe("http");
    expect(protocolForHost("localhost:3002")).toBe("http");
  });

  it("is http for a tenant subdomain of localhost — the bug this fixes: a naive " +
    "startsWith('localhost') check misses this, since the Host header here is 'grace.localhost', " +
    "not 'localhost'", () => {
    expect(protocolForHost("grace.localhost:3002")).toBe("http");
    expect(protocolForHost("grace.localhost")).toBe("http");
  });

  it("is http for 127.0.0.1, with or without a port", () => {
    expect(protocolForHost("127.0.0.1:3002")).toBe("http");
  });

  it("is https for a real domain, apex or tenant subdomain alike", () => {
    expect(protocolForHost("openfaithmap.org")).toBe("https");
    expect(protocolForHost("grace.openfaithmap.org")).toBe("https");
  });
});

describe("injectSitesSegment", () => {
  it("inserts _sites/{slug} right after the locale segment", () => {
    expect(injectSitesSegment("/en", "grace")).toBe("/en/_sites/grace");
    expect(injectSitesSegment("/en/about", "grace")).toBe("/en/_sites/grace/about");
    expect(injectSitesSegment("/uk/events/christmas", "grace")).toBe("/uk/_sites/grace/events/christmas");
  });
});
