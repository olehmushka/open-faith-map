import { getTranslations } from "next-intl/server";

import { auth } from "@/auth";
import { EXCLUDED_TAXON_CODES } from "@/lib/dictionaries";
import { oikumenea } from "@/lib/oikumenea";
import { submitRegistration } from "@/lib/registration";
import { redirect } from "@/i18n/navigation";

// go-oikumenea's own locale codes are ISO 639-3; this app's URL-facing locales are ISO 639-1.
const OIKUMENEA_LOCALE: Record<string, string> = { en: "eng", uk: "ukr", es: "spa", pt: "por" };

export default async function RegisterPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  const t = await getTranslations("RegisterPage");
  const { error } = await searchParams;
  const oikumeneaLocale = OIKUMENEA_LOCALE[locale] ?? "eng";

  const client = await oikumenea();
  const [taxaPage, countries] = await Promise.all([
    client.religion.listTaxa(undefined, undefined, undefined, undefined, undefined, 500),
    client.geo.listCountries(),
  ]);

  const taxa = taxaPage.taxa
    .filter((taxon) => !EXCLUDED_TAXON_CODES.has(taxon.code))
    .map((taxon) => ({
      id: taxon.id,
      name: taxon.name[oikumeneaLocale] ?? taxon.name["eng"] ?? Object.values(taxon.name)[0] ?? taxon.code,
    }))
    .sort((a, b) => a.name.localeCompare(b.name));

  const countryOptions = countries.countries
    .map((c) => ({ id: c.id, name: c.name[oikumeneaLocale] ?? c.name["eng"] ?? Object.values(c.name)[0] ?? c.code }))
    .sort((a, b) => a.name.localeCompare(b.name));

  async function submit(formData: FormData) {
    "use server";
    const taxonId = String(formData.get("taxonId") ?? "");
    const congregationName = String(formData.get("congregationName") ?? "");
    const countryId = String(formData.get("countryId") ?? "");
    const locality = String(formData.get("locality") ?? "") || undefined;
    const street = String(formData.get("street") ?? "") || undefined;
    const latitude = Number(formData.get("latitude"));
    const longitude = Number(formData.get("longitude"));

    try {
      const req = await submitRegistration({
        taxonId,
        congregationName,
        countryId,
        locality,
        street,
        coordinate: { latitude, longitude },
      });
      redirect({ href: `/register/submitted?id=${req.id}`, locale });
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        redirect({
          href: `/register?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`,
          locale,
        });
      }
      throw e;
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{t("heading")}</h1>
      <p className="text-sm">{t("intro")}</p>

      {error && (
        <p className="rounded border border-red-500 p-3 text-sm">
          {error === "Registration:TaxonExcluded" ? t("errorTaxonExcluded") : t("errorGeneric", { error })}
        </p>
      )}

      <form action={submit} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("traditionLabel")}</span>
          <select name="taxonId" required className="rounded border px-3 py-2">
            <option value="">{t("traditionPlaceholder")}</option>
            {taxa.map((taxon) => (
              <option key={taxon.id} value={taxon.id}>
                {taxon.name}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("congregationNameLabel")}</span>
          <input name="congregationName" required className="rounded border px-3 py-2" />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("countryLabel")}</span>
          <select name="countryId" required className="rounded border px-3 py-2">
            <option value="">{t("countryPlaceholder")}</option>
            {countryOptions.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("cityLabel")}</span>
          <input name="locality" className="rounded border px-3 py-2" />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{t("streetLabel")}</span>
          <input name="street" className="rounded border px-3 py-2" />
        </label>

        <div className="grid grid-cols-2 gap-4">
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">{t("latitudeLabel")}</span>
            <input name="latitude" type="number" step="any" required className="rounded border px-3 py-2" />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">{t("longitudeLabel")}</span>
            <input name="longitude" type="number" step="any" required className="rounded border px-3 py-2" />
          </label>
        </div>

        <button type="submit" className="rounded border px-4 py-2">
          {t("submit")}
        </button>
      </form>
    </main>
  );
}
