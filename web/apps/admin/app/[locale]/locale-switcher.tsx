"use client";

import { useLocale, useTranslations } from "next-intl";

import { routing } from "@/i18n/routing";
import { usePathname, useRouter } from "@/i18n/navigation";

export function LocaleSwitcher() {
  const locale = useLocale();
  const pathname = usePathname();
  const router = useRouter();
  const t = useTranslations("LocaleSwitcher");

  return (
    <label className="flex items-center gap-1 text-xs text-gray-500">
      <span className="hidden sm:inline">{t("label")}</span>
      <select
        aria-label={t("label")}
        value={locale}
        onChange={(e) => router.replace(pathname, { locale: e.target.value })}
        className="rounded border px-2 py-1 text-sm"
      >
        {routing.locales.map((l) => (
          <option key={l} value={l}>
            {t(l)}
          </option>
        ))}
      </select>
    </label>
  );
}
