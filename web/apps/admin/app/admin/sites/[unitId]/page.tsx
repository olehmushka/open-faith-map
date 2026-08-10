import Link from "next/link";
import { redirect } from "next/navigation";

import { auth } from "@/auth";
import { createSite, getSite, updateSiteTheme } from "@/lib/content";

// Site creation + theme editor for one congregation's unit (M3, docs/modules/content.md). getSite
// is a public-read call (ContentPublicService) — no site yet is a normal, expected state (a
// congregation admin's first visit), not an error, hence .catch(() => null) rather than surfacing
// Content:SiteNotFound.
export default async function SitePage({ params }: { params: Promise<{ unitId: string }> }) {
  const session = await auth();
  if (!session) redirect("/login");

  const { unitId } = await params;
  const site = await getSite(unitId).catch(() => null);

  if (!site) {
    async function create(formData: FormData) {
      "use server";
      const slug = String(formData.get("slug") ?? "");
      try {
        await createSite({ congregationUnitId: unitId, slug });
      } catch (e) {
        if (e && typeof e === "object" && "errorName" in e) {
          redirect(`/admin/sites/${unitId}?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`);
        }
        throw e;
      }
      redirect(`/admin/sites/${unitId}`);
    }

    return (
      <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6 py-12">
        <h1 className="text-2xl font-semibold">Create your site</h1>
        <p className="text-sm">
          Choose a slug — this is the public path your site will be reachable at. Slugs are
          first-come, first-served.
        </p>
        <form action={create} className="flex flex-col gap-4">
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">Slug</span>
            <input name="slug" required pattern="[a-z0-9-]+" className="rounded border px-3 py-2" />
          </label>
          <button type="submit" className="rounded border px-4 py-2">
            Create site
          </button>
        </form>
      </main>
    );
  }

  const theme = (site.theme ?? {}) as { accentColor?: string; fontPairing?: string; headerLayout?: string };

  async function saveTheme(formData: FormData) {
    "use server";
    await updateSiteTheme(site!.id, {
      accentColor: String(formData.get("accentColor") ?? ""),
      fontPairing: String(formData.get("fontPairing") ?? ""),
      headerLayout: String(formData.get("headerLayout") ?? ""),
    });
    redirect(`/admin/sites/${unitId}`);
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">Site: {site.slug}</h1>
      <Link href={`/admin/sites/${unitId}/documents`} className="underline">
        Manage pages
      </Link>

      <section>
        <h2 className="text-lg font-medium">Theme</h2>
        <form action={saveTheme} className="mt-2 flex flex-col gap-4">
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">Accent color</span>
            <input name="accentColor" defaultValue={theme.accentColor ?? ""} className="rounded border px-3 py-2" />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">Font pairing</span>
            <input name="fontPairing" defaultValue={theme.fontPairing ?? ""} className="rounded border px-3 py-2" />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">Header layout</span>
            <input name="headerLayout" defaultValue={theme.headerLayout ?? ""} className="rounded border px-3 py-2" />
          </label>
          <button type="submit" className="rounded border px-4 py-2">
            Save theme
          </button>
        </form>
      </section>
    </main>
  );
}
