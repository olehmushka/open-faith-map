import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin();

const nextConfig: NextConfig = {
  // Needed for the compose-service Dockerfile's copy-only runtime stage (M1).
  output: "standalone",
};

export default withNextIntl(nextConfig);
