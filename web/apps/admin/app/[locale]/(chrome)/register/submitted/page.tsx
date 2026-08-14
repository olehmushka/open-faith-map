import { getTranslations } from "next-intl/server";

import { getRegistration } from "@/lib/registration";
import { Link } from "@/i18n/navigation";

export default async function RegisterSubmittedPage({
  searchParams,
}: {
  searchParams: Promise<{ id?: string }>;
}) {
  const t = await getTranslations("RegisterSubmittedPage");
  const { id } = await searchParams;
  const req = id ? await getRegistration(id).catch(() => null) : null;

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-4 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      {req ? (
        <p>
          {t.rich("pendingWithName", {
            name: req.congregationName,
            status: req.status,
            strong: (chunks) => <strong>{chunks}</strong>,
          })}
        </p>
      ) : (
        <p>{t("pendingGeneric")}</p>
      )}
      <Link href="/" className="underline">
        {t("backHome")}
      </Link>
    </main>
  );
}
