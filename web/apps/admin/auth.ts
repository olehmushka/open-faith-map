// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// openfaithmap-admin's session layer — the ONLY session anywhere in OpenFaithMap (M1, moved here
// from openfaithmap-web at M2.1; docs/modules/web-admin.md, D-AdminSurface). Google is the sole
// OIDC provider — no Keycloak (docs/architecture/decisions.md's D-GoogleDirect): go-oikumenea
// trusts Google directly (deploy/oikumenea-install.yml), so this is the simple, single-provider
// version of go-oikumenea's own multi-IdP console-bff pattern (docs/web-ui.md in the go-oikumenea
// repo).
//
// Forwards the Google ID token, not the access token, as the bearer on every go-oikumenea call —
// the access token is an opaque string go-oikumenea can't verify; the ID token is the JWT whose
// `aud` go-oikumenea's Google issuer entry pins.
//
// M10.2 (D-DirectTokenVerification's amendment): the ID token expires in 1 hour but the admin
// session lives far longer, and the jwt callback only receives a populated `account` at initial
// sign-in — so without a refresh, every session silently forwards an expired token past the
// one-hour mark. access_type/prompt below force Google to issue a refresh_token on every sign-in
// (Google otherwise only grants one on first consent), and the jwt callback refreshes the ID token
// once its own expiry is close, on every subsequent call.
//
// M11.3 (D-SessionTracking): the Google ID token above is signed by Google, so a custom sessionId
// claim can't be injected into it — the backend never even sees this file's own JWT (its encrypted
// session cookie), only the forwarded ID token. So sessionId here travels separately: stamped onto
// THIS cookie JWT at sign-in, then sent as its own X-Session-Id header on every API call
// (lib/core.ts's client()), alongside — not instead of — the existing bearer.
import NextAuth, { type DefaultSession } from "next-auth";
import Google from "next-auth/providers/google";
import { headers } from "next/headers";

declare module "next-auth" {
  interface Session {
    /** The Google ID token forwarded as the bearer on openfaithmap-api calls (lib/openfaithmap). */
    idToken?: string;
    /** Set when the refresh attempt itself failed — the session should be treated as ended. */
    error?: "RefreshTokenError";
    /** This browser session's own identity_sessions row id (M11.3) — sent as X-Session-Id. */
    sessionId?: string;
    user: DefaultSession["user"];
  }
}

declare module "@auth/core/jwt" {
  interface JWT {
    idToken?: string;
    refreshToken?: string;
    /** ID token expiry, epoch seconds. */
    expiresAt?: number;
    error?: "RefreshTokenError";
    /** M11.3 — set once, at initial sign-in; never rotated on refresh. */
    sessionId?: string;
  }
}

/** Best-effort User-Agent for identity_sessions.device_label — never throws. */
async function currentUserAgent(): Promise<string | undefined> {
  try {
    return (await headers()).get("user-agent") ?? undefined;
  } catch {
    return undefined;
  }
}

/** Refreshes the ID token via Google's token endpoint. Throws on any failure. */
async function refreshIdToken(refreshToken: string) {
  const res = await fetch("https://oauth2.googleapis.com/token", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      client_id: process.env.AUTH_GOOGLE_ID!,
      client_secret: process.env.AUTH_GOOGLE_SECRET!,
      grant_type: "refresh_token",
      refresh_token: refreshToken,
    }),
  });
  const body = await res.json();
  if (!res.ok) throw new Error(`refresh failed: ${res.status} ${JSON.stringify(body)}`);
  return body as { id_token: string; expires_in: number; refresh_token?: string };
}

/**
 * Creates the identity_sessions row backing this sign-in (M11.3, CoreService.registerSession) and
 * returns its id. Called via a raw fetch, not lib/core.ts's client() — that helper itself calls
 * auth() to read the very session being constructed here, which would be circular. Throws on any
 * failure: a sign-in this app can't register a session for would 401 on every subsequent API call
 * anyway, so failing the sign-in itself (matching refreshIdToken's own throw-on-failure contract
 * above) is preferable to silently minting a session nothing can use.
 */
async function registerSessionOnBackend(idToken: string): Promise<string> {
  const baseUrl = process.env.OPENFAITHMAP_API_BASE_URL?.trim().replace(/\/+$/, "");
  if (!baseUrl) throw new Error("OPENFAITHMAP_API_BASE_URL is not set.");
  const res = await fetch(`${baseUrl}/core/v1/sessions`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${idToken}` },
    body: JSON.stringify({ deviceLabel: await currentUserAgent() }),
  });
  const body = await res.json();
  if (!res.ok) throw new Error(`registerSession failed: ${res.status} ${JSON.stringify(body)}`);
  return (body as { id: string }).id;
}

export const { handlers, signIn, signOut, auth } = NextAuth({
  trustHost: true,
  providers: [
    Google({
      clientId: process.env.AUTH_GOOGLE_ID,
      clientSecret: process.env.AUTH_GOOGLE_SECRET,
      // access_type=offline is what makes Google issue a refresh_token at all; prompt=consent
      // forces it on every sign-in rather than only the first (Google otherwise omits it on a
      // returning user's silent re-auth), so a revoked-then-re-granted consent still gets one.
      authorization: { params: { scope: "openid email profile", access_type: "offline", prompt: "consent" } },
    }),
  ],
  callbacks: {
    async jwt({ token, account }) {
      if (account) {
        // Initial sign-in: account is populated exactly once.
        token.idToken = account.id_token;
        token.refreshToken = account.refresh_token;
        token.expiresAt = account.expires_at; // epoch seconds, from Google's id_token exp claim
        // M11.3: register the backing identity_sessions row now, before this token is ever used as
        // a bearer — every other endpoint requires a valid X-Session-Id from the first call on.
        if (account.id_token) {
          token.sessionId = await registerSessionOnBackend(account.id_token);
        }
        return token;
      }

      if (!token.expiresAt || Date.now() < token.expiresAt * 1000) {
        return token; // still valid
      }
      if (!token.refreshToken) {
        return { ...token, idToken: undefined, error: "RefreshTokenError" as const };
      }
      try {
        const refreshed = await refreshIdToken(token.refreshToken);
        token.idToken = refreshed.id_token;
        token.expiresAt = Math.floor(Date.now() / 1000) + refreshed.expires_in;
        // Google does not always return a new refresh_token; keep the existing one when absent.
        if (refreshed.refresh_token) token.refreshToken = refreshed.refresh_token;
        delete token.error;
      } catch {
        // Refresh failed (revoked, expired, network) — clear the token rather than forward a
        // stale/garbage one; the session callback below surfaces this as session.error.
        token.idToken = undefined;
        token.error = "RefreshTokenError";
      }
      return token;
    },
    async session({ session, token }) {
      session.idToken = token.idToken;
      session.error = token.error;
      session.sessionId = token.sessionId;
      return session;
    },
  },
  pages: { signIn: "/login" },
});
