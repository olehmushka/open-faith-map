import { getTranslations } from "next-intl/server";
import { LogOut, MapPinned, ShieldCheck, User } from "lucide-react";

import { auth, signOut } from "@/auth";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

// The front door — previously there was no page at all for the bare "/{locale}" URL (a visitor
// landed on a 404). Signed-out visitors get a single sign-in CTA; signed-in visitors get a
// prominent link into the admin console plus the same quick links whoami/page.tsx already offers,
// so this page can act as a proper home rather than just a login gate.
export default async function HomePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const session = await auth();
  const t = await getTranslations("HomePage");

  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col items-center justify-center gap-10 px-6 text-center">
      <div className="flex flex-col gap-3">
        <h1 className="text-3xl font-semibold tracking-tight">{t("heading")}</h1>
        <p className="text-muted-foreground">{t("tagline")}</p>
      </div>

      {session ? (
        <div className="grid w-full gap-4 sm:grid-cols-2">
          <Card className="text-left">
            <CardHeader>
              <MapPinned className="size-5 text-muted-foreground" />
              <CardTitle className="text-base">{t("adminHeading")}</CardTitle>
              <CardDescription>{t("adminDescription")}</CardDescription>
            </CardHeader>
            <CardContent>
              <Button asChild className="w-full">
                <Link href="/admin/congregation-import">{t("openAdmin")}</Link>
              </Button>
            </CardContent>
          </Card>

          <Card className="text-left">
            <CardHeader>
              <User className="size-5 text-muted-foreground" />
              <CardTitle className="text-base">{t("accountHeading")}</CardTitle>
              <CardDescription>{t("accountDescription")}</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-2">
              <Button asChild variant="outline" className="w-full">
                <Link href="/whoami">{t("whoami")}</Link>
              </Button>
              <Button asChild variant="outline" className="w-full">
                <Link href="/my-congregation">{t("myCongregation")}</Link>
              </Button>
            </CardContent>
          </Card>
        </div>
      ) : (
        <div className="flex flex-col items-center gap-4">
          <ShieldCheck className="size-8 text-muted-foreground" />
          <Button asChild size="lg">
            <Link href="/login">{t("signIn")}</Link>
          </Button>
        </div>
      )}

      {session && (
        <form
          action={async () => {
            "use server";
            await signOut({ redirectTo: `/${locale}` });
          }}
        >
          <Button type="submit" variant="ghost" size="sm" className="text-muted-foreground">
            <LogOut className="size-3.5" />
            {t("signOut")}
          </Button>
        </form>
      )}
    </main>
  );
}
