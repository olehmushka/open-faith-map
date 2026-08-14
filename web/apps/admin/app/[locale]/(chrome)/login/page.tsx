import { getTranslations } from "next-intl/server";

import { signIn } from "@/auth";

export default async function LoginPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations("LoginPage");
  return (
    <main className="mx-auto flex min-h-screen max-w-sm flex-col justify-center gap-4 px-6">
      <h1 className="text-xl font-semibold">{t("heading")}</h1>
      <form
        action={async () => {
          "use server";
          await signIn("google", { redirectTo: `/${locale}/whoami` });
        }}
      >
        <button type="submit" className="rounded border px-4 py-2">
          {t("googleButton")}
        </button>
      </form>
    </main>
  );
}
