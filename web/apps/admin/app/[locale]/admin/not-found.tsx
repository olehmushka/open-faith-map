import { getTranslations } from "next-intl/server";
import { FileQuestion } from "lucide-react";

import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";

// Rendered inside admin/layout.tsx's shell (sidebar + topbar stay visible) whenever notFound() is
// called from within /admin/** — either explicitly (e.g. an unknown documentId) or implicitly via
// the [...catchAll] route for a URL with no matching page at all.
export default async function AdminNotFound() {
  const t = await getTranslations("AdminNotFoundPage");

  return (
    <div className="mx-auto flex w-full max-w-md flex-col items-center gap-4 py-24 text-center">
      <FileQuestion className="size-10 text-muted-foreground" />
      <h1 className="text-xl font-semibold">{t("heading")}</h1>
      <p className="text-sm text-muted-foreground">{t("description")}</p>
      <Button asChild>
        <Link href="/admin/congregation-import">{t("backLink")}</Link>
      </Button>
    </div>
  );
}
