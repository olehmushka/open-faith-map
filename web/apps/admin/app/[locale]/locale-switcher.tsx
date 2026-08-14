"use client";

import { useLocale, useTranslations } from "next-intl";

import { routing } from "@/i18n/routing";
import { usePathname, useRouter } from "@/i18n/navigation";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export function LocaleSwitcher() {
  const locale = useLocale();
  const pathname = usePathname();
  const router = useRouter();
  const t = useTranslations("LocaleSwitcher");

  return (
    <Select
      value={locale}
      onValueChange={(value) => router.replace(pathname, { locale: value })}
    >
      <SelectTrigger aria-label={t("label")} size="sm" className="w-auto">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {routing.locales.map((l) => (
          <SelectItem key={l} value={l}>
            {t(l)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
