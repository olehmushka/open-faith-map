import { notFound } from "next/navigation";

// Exists purely so an unmatched /admin/** path has a real route to match. Without this, Next.js
// can't find ANY matching segment tree for e.g. /admin/typo-of-a-real-section, so it falls back to
// the app-level not-found — skipping admin/layout.tsx entirely, which means the session check
// there never runs and an unauthenticated visitor never gets redirected to /login. Matching this
// catch-all still renders admin/layout.tsx first (so the redirect-to-login still fires for anyone
// without a session), then notFound() renders admin/not-found.tsx for anyone who IS logged in.
export default function AdminCatchAll() {
  notFound();
}
