import { getTranslations } from "next-intl/server";
import { FileQuestion } from "lucide-react";

import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";

// Catches any top-level path that isn't /admin/**, /login, /register, /whoami, or
// /my-congregation (see the sibling [...catchAll]/page.tsx). Rendered inside app/[locale]/layout.tsx
// only — no sidebar/topbar here, since a visitor hitting this may not have a session at all.
export default async function LocaleNotFound() {
  const t = await getTranslations("NotFoundPage");

  return (
    <div className="mx-auto flex min-h-screen max-w-sm flex-col items-center justify-center gap-4 px-6 text-center">
      <FileQuestion className="size-10 text-muted-foreground" />
      <h1 className="text-xl font-semibold">{t("heading")}</h1>
      <p className="text-sm text-muted-foreground">{t("description")}</p>
      <Button asChild>
        <Link href="/login">{t("backLink")}</Link>
      </Button>
    </div>
  );
}
