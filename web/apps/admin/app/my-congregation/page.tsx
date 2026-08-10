import Link from "next/link";
import { redirect } from "next/navigation";

import { oikumenea } from "@/lib/oikumenea";
import { listRegistrations } from "@/lib/registration";

// The M2 "see their own roster" exit criterion. Finds the caller's own most recent APPROVED
// registration (listRegistrations may return every request if the caller happens to also be an
// operator, so filter to their own submissions explicitly rather than assuming scoping) and renders
// membership.listMembers over its unit.
export default async function MyCongregationPage() {
  const client = await oikumenea();
  const who = await client.identityFederation.whoami().catch(() => null);
  if (!who) redirect("/login");

  const { requests } = await listRegistrations();
  const mine = requests
    .filter((r) => r.submittedByPersonId === who.personId && r.status === "APPROVED" && r.createdUnitId)
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));

  if (mine.length === 0) {
    return (
      <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-4 px-6 py-12">
        <h1 className="text-2xl font-semibold">My congregation</h1>
        <p>You don&apos;t have an approved congregation yet.</p>
        <Link href="/register" className="underline">
          Register a congregation
        </Link>
      </main>
    );
  }

  const unitId = mine[0].createdUnitId!;
  const [unit, memberPage] = await Promise.all([
    client.tenant.getUnit(unitId),
    client.membership.listMembers(unitId),
  ]);

  const members = await Promise.all(
    memberPage.memberships.map(async (m) => {
      const person = await client.person.getPerson(m.personId).catch(() => null);
      return { ...m, displayName: person?.displayName ?? m.personId };
    }),
  );

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">{mine[0].congregationName}</h1>
      <p className="text-sm">Unit status: {unit.state}</p>
      <Link href={`/admin/sites/${unitId}`} className="underline">
        Manage site
      </Link>

      <section>
        <h2 className="text-lg font-medium">Roster</h2>
        {members.length === 0 ? (
          <p className="text-sm">No members yet.</p>
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
