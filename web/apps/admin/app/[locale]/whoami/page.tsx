import { getTranslations } from "next-intl/server";

import { auth, signOut } from "@/auth";
import { oikumenea } from "@/lib/oikumenea";
import { Link, redirect } from "@/i18n/navigation";

// The M1 exit-criterion proof artifact (docs/milestones.md): calls a real go-oikumenea endpoint
// with the logged-in user's forwarded Google ID token, proving the session layer's token passthrough
// works end-to-end, not just that login itself succeeds.
export default async function WhoamiPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const session = await auth();
  if (!session) {
    redirect({ href: "/login", locale });
  }

  const t = await getTranslations("WhoamiPage");
  const client = await oikumenea();
  const who = await client.identityFederation.whoami();

  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col justify-center gap-4 px-6">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p>{t.rich("description", { code: (chunks) => <code>{chunks}</code> })}</p>
      <pre className="overflow-x-auto rounded border p-4">{JSON.stringify(who, null, 2)}</pre>

      <nav className="flex flex-col gap-2 text-sm">
        <Link href="/register" className="underline">
          {t("registerLink")}
        </Link>
        <Link href="/admin/registrations" className="underline">
          {t("reviewLink")}
        </Link>
        <Link href="/my-congregation" className="underline">
          {t("myCongregationLink")}
        </Link>
      </nav>

      <form
        action={async () => {
          "use server";
          await signOut({ redirectTo: `/${locale}` });
        }}
      >
        <button type="submit" className="rounded border px-4 py-2">
          {t("signOut")}
        </button>
      </form>
    </main>
  );
}
