import { redirect } from "next/navigation";

import { auth } from "@/auth";
import { oikumenea } from "@/lib/oikumenea";
import { submitRegistration } from "@/lib/registration";

// D-Exclusions (architecture/decisions.md), same codes as
// internal/registration/domain.ExcludedTaxonCodes — the authoritative check runs server-side in
// openfaithmap-api regardless; filtering them out of the picker is a UX nicety only.
const EXCLUDED_TAXON_CODES = new Set(["russian_orthodox_church", "jehovahs_witnesses", "lds_church"]);

export default async function RegisterPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const session = await auth();
  if (!session) redirect("/login");

  const { error } = await searchParams;

  const client = await oikumenea();
  const [taxaPage, countries] = await Promise.all([
    client.religion.listTaxa(undefined, undefined, undefined, undefined, undefined, 500),
    client.geo.listCountries(),
  ]);

  const taxa = taxaPage.taxa
    .filter((t) => !EXCLUDED_TAXON_CODES.has(t.code))
    .map((t) => ({ id: t.id, name: t.name["eng"] ?? Object.values(t.name)[0] ?? t.code }))
    .sort((a, b) => a.name.localeCompare(b.name));

  const countryOptions = countries.countries
    .map((c) => ({ id: c.id, name: c.name["eng"] ?? Object.values(c.name)[0] ?? c.code }))
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
      redirect(`/register/submitted?id=${req.id}`);
    } catch (e) {
      if (e && typeof e === "object" && "errorName" in e) {
        redirect(`/register?error=${encodeURIComponent(String((e as { errorName: string }).errorName))}`);
      }
      throw e;
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">Register your congregation</h1>
      <p className="text-sm">
        Submitted requests are reviewed by a registration operator before your congregation appears
        on the map.
      </p>

      {error && (
        <p className="rounded border border-red-500 p-3 text-sm">
          {error === "Registration:TaxonExcluded"
            ? "That tradition is not eligible for registration on OpenFaithMap."
            : `Something went wrong: ${error}`}
        </p>
      )}

      <form action={submit} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Tradition</span>
          <select name="taxonId" required className="rounded border px-3 py-2">
            <option value="">Select a tradition…</option>
            {taxa.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Congregation name</span>
          <input name="congregationName" required className="rounded border px-3 py-2" />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Country</span>
          <select name="countryId" required className="rounded border px-3 py-2">
            <option value="">Select a country…</option>
            {countryOptions.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">City</span>
          <input name="locality" className="rounded border px-3 py-2" />
        </label>

        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">Street</span>
          <input name="street" className="rounded border px-3 py-2" />
        </label>

        <div className="grid grid-cols-2 gap-4">
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">Latitude</span>
            <input name="latitude" type="number" step="any" required className="rounded border px-3 py-2" />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium">Longitude</span>
            <input name="longitude" type="number" step="any" required className="rounded border px-3 py-2" />
          </label>
        </div>

        <button type="submit" className="rounded border px-4 py-2">
          Submit for review
        </button>
      </form>
    </main>
  );
}
