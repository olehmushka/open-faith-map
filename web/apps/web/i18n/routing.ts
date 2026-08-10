import { defineRouting } from "next-intl/routing";

export const routing = defineRouting({
  locales: ["en", "uk", "es", "pt"],
  defaultLocale: "en",
  localePrefix: "always",
});
