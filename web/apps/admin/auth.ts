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
import NextAuth, { type DefaultSession } from "next-auth";
import Google from "next-auth/providers/google";

declare module "next-auth" {
  interface Session {
    /** The Google ID token forwarded as the bearer on go-oikumenea calls (lib/oikumenea.ts). */
    idToken?: string;
    /** Set when the refresh attempt itself failed — the session should be treated as ended. */
    error?: "RefreshTokenError";
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
      return session;
    },
  },
  pages: { signIn: "/login" },
});
