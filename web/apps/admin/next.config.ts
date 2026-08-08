import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Needed for the compose-service Dockerfile's copy-only runtime stage (M1).
  output: "standalone",
};

export default nextConfig;
