import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin();

// D-PublicSiteCSP (M14.1): a static (non-nonce) CSP plus baseline security headers on every
// response. This is not the primary defense against the stored-XSS class this milestone closes —
// that's the write/render-time URL-scheme allowlist (blockvalidation.go / lib/block-security.ts).
// It's belt-and-braces against everything else an unhardened response is otherwise exposed to.
// script-src/style-src need 'unsafe-inline' because Next's App Router streams RSC payloads and
// hydration data via inline <script>/<style> tags with no nonce wired up here (a nonce would
// require per-request dynamic rendering via proxy.ts, fighting M14.17's later cache-invalidation
// goal — a deliberate scoping call, not an oversight). img-src stays scheme-broad (https:) rather
// than a host allowlist: M14.3's Drive/Dropbox/OneDrive URL normalizer hasn't shipped yet, so
// admins can currently point image.url at any https host.
const CSP = [
  "default-src 'self'",
  "script-src 'self' 'unsafe-inline'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' https: data:",
  "font-src 'self' data:",
  "connect-src 'self'",
  "frame-src 'self' https://www.youtube.com",
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'self'",
].join("; ");

const nextConfig: NextConfig = {
  // Needed for the compose-service Dockerfile's copy-only runtime stage (M1).
  output: "standalone",
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          { key: "Content-Security-Policy", value: CSP },
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
        ],
      },
    ];
  },
};

export default withNextIntl(nextConfig);
