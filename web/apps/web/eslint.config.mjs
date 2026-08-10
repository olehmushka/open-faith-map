import nextPlugin from "@next/eslint-plugin-next";
import tseslint from "typescript-eslint";

// Native flat config, built directly from @next/eslint-plugin-next + typescript-eslint rather
// than routing through eslint-config-next's legacy shareable config via FlatCompat — that bridge
// currently throws on eslint-plugin-react's self-referential flat config object
// (https://github.com/vercel/next.js/issues, eslint-config-next 16.3.0 + eslint 9/10). Revisit
// once upstream fixes land; this gets the same @next/next rule set without the breakage.
export default tseslint.config(
  {
    // lib/openfaithmap/generated is GENERATED from the Conjure contract (scripts/gen-ts-client.sh,
    // M4) — never hand-edited, so never linted; same reasoning as .next/** below.
    ignores: [".next/**", "node_modules/**", "lib/openfaithmap/generated/**"],
  },
  tseslint.configs.recommended,
  {
    plugins: {
      "@next/next": nextPlugin,
    },
    rules: {
      ...nextPlugin.configs["core-web-vitals"].rules,
    },
  },
);
