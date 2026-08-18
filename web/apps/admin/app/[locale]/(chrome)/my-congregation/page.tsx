import { getTranslations } from "next-intl/server";

import { getPersons, getUnit, listMembershipsByUnit, whoami } from "@/lib/core";
import { listRegistrations } from "@/lib/registration";
import { Link, redirect } from "@/i18n/navigation";

// The M2 "see their own roster" exit criterion. Finds the caller's own most recent APPROVED
// registration (listRegistrations may return every request if the caller happens to also be an
// operator, so filter to their own submissions explicitly rather than assuming scoping) and renders
// the unit's membership roster. M10.7: repointed from go-oikumenea (lib/oikumenea.ts, deleted this
// milestone) to lib/core.ts; the per-member getPerson loop is now one batched getPersons call.
export default async function MyCongregationPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  const t = await getTranslations("MyCongregationPage");
  const who = await whoami().catch(() => null);
  if (!who) return redirect({ href: "/login", locale });

  const { requests } = await listRegistrations();
  const mine = requests
    .filter((r) => r.submittedByPersonId === who.personId && r.status === "APPROVED" && r.createdUnitId)
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));

  if (mine.length === 0) {
    return (
      <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-4 px-6 py-12">
        <h1 className="text-2xl font-semibold">{t("heading")}</h1>
        <p>{t("noCongregation")}</p>
        <Link href="/register" className="underline">
          {t("registerLink")}
        </Link>
      </main>
    );
  }

  const unitId = mine[0].createdUnitId!;
  const [unit, memberships] = await Promise.all([getUnit(unitId), listMembershipsByUnit(unitId)]);

  const persons = await getPersons(memberships.map((m) => m.personId));
  const personById = new Map(persons.map((p) => [p.id, p]));
  const members = memberships.map((m) => ({
    ...m,
    displayName: personById.get(m.personId)?.displayName ?? m.personId,
  }));

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{mine[0].congregationName}</h1>
      <p className="text-sm">{t("unitStatus", { status: unit.state })}</p>
      <Link href={`/admin/sites/${unitId}`} className="underline">
        {t("manageSite")}
      </Link>

      <section>
        <h2 className="text-lg font-medium">{t("rosterHeading")}</h2>
        {members.length === 0 ? (
          <p className="text-sm">{t("noMembers")}</p>
        ) : (
          <ul className="mt-2 flex flex-col gap-2">
            {members.map((m) => (
              <li key={m.id} className="rounded border p-3 text-sm">
                {m.displayName} — {m.status}
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
