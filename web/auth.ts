// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// openfaithmap-web's session layer (M1, docs/modules/web-facade.md). Google is the sole OIDC
// provider — no Keycloak (docs/architecture/decisions.md's D-CoreDependency): go-oikumenea trusts
// Google directly (deploy/oikumenea-install.yml), so this is the simple, single-provider version of
// go-oikumenea's own multi-IdP console-bff pattern (docs/web-ui.md in the go-oikumenea repo).
//
// Forwards the Google ID token, not the access token, as the bearer on every go-oikumenea call —
// the access token is an opaque string go-oikumenea can't verify; the ID token is the JWT whose
// `aud` go-oikumenea's Google issuer entry pins.
import NextAuth, { type DefaultSession } from "next-auth";
import Google from "next-auth/providers/google";

declare module "next-auth" {
  interface Session {
    /** The Google ID token forwarded as the bearer on go-oikumenea calls (web/lib/oikumenea.ts). */
    idToken?: string;
    user: DefaultSession["user"];
  }
}

declare module "@auth/core/jwt" {
  interface JWT {
    idToken?: string;
  }
}

export const { handlers, signIn, signOut, auth } = NextAuth({
  trustHost: true,
  providers: [
    Google({
      clientId: process.env.AUTH_GOOGLE_ID,
      clientSecret: process.env.AUTH_GOOGLE_SECRET,
      authorization: { params: { scope: "openid email profile" } },
    }),
  ],
  callbacks: {
    async jwt({ token, account }) {
      if (account?.id_token) token.idToken = account.id_token;
      return token;
    },
    async session({ session, token }) {
      session.idToken = token.idToken;
      return session;
    },
  },
  pages: { signIn: "/login" },
});
