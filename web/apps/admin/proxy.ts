import createMiddleware from "next-intl/middleware";

import { routing } from "./i18n/routing";

export default createMiddleware(routing);

// M11.9 found a real bug in this matcher: "api" with no trailing-slash boundary excludes any path
// merely STARTING with those letters, not just the intended "/api/*" Next.js route handlers (e.g.
// NextAuth's /api/auth/[...nextauth]) — which silently also excluded the new /api-keys page from
// locale-prefix middleware, 404ing on a bare (unprefixed) request. "api/" (with the slash) only
// matches the literal /api/... prefix.
export const config = {
  matcher: ["/((?!api/|_next|_vercel|.*\\..*).*)"],
};
