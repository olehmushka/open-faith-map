// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.5: this app's first test runner. Offline component tests only — no server, no auth — added
// because there is no headless Google OAuth login path for openfaithmap-admin (per M14.4's own
// verification notes), so interactive logic like drag/keyboard reordering has no browser-proof path
// through the running stack the way the public site's renderer does.
import path from "node:path";
import { fileURLToPath } from "node:url";

import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

const dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": dirname,
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    globals: true,
  },
});
