import { redirect } from "next/navigation";

import { auth } from "@/auth";
import { createJurisdictionUnit, searchJurisdictionUnits } from "@/lib/jurisdiction";
import { approveRegistration, listRegistrations, rejectRegistration } from "@/lib/registration";

// Renders whatever listRegistrations returns for the caller — openfaithmap-api itself decides
// operator (all requests) vs. submitter (their own only) by asking go-oikumenea's PDP live
// (MyCapabilities), never a locally-cached role (D-Facade). A non-operator lands here and simply
// sees their own submissions; approve/reject actions still go through the real PDP check
// regardless of what this page renders (web-facade.md's "no client-side authorization").
export default async function RegistrationsPage({
  searchParams,
}: {
  searchParams: Promise<{ jurisdictionQuery?: string }>;
}) {
  const session = await auth();
  if (!session) redirect("/login");

  const { requests } = await listRegistrations();
  const { jurisdictionQuery } = await searchParams;
  const jurisdictionResults = jurisdictionQuery ? await searchJurisdictionUnits(jurisdictionQuery) : [];

  async function approve(formData: FormData) {
    "use server";
    const jurisdictionUnitId = String(formData.get("jurisdictionUnitId") ?? "").trim() || undefined;
    await approveRegistration(String(formData.get("id")), undefined, jurisdictionUnitId);
    redirect("/admin/registrations");
  }

  async function createJurisdiction(formData: FormData) {
    "use server";
    const parentUnitId = String(formData.get("parentUnitId") ?? "").trim() || undefined;
    const code = String(formData.get("code") ?? "").trim();
    const name = String(formData.get("name") ?? "").trim();
    if (!code || !name) return;
    const rootUnitId = process.env.REGISTRATION_ROOT_UNIT_ID;
    if (!rootUnitId) throw new Error("REGISTRATION_ROOT_UNIT_ID is not set.");
    const unit = await createJurisdictionUnit(parentUnitId ?? rootUnitId, code, name);
    redirect(`/admin/registrations?jurisdictionQuery=${encodeURIComponent(unit.name)}`);
  }

  async function reject(formData: FormData) {
    "use server";
    const reason = String(formData.get("reason") ?? "").trim();
    if (!reason) return;
    await rejectRegistration(String(formData.get("id")), reason);
    redirect("/admin/registrations");
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">Registration requests</h1>

      <section className="rounded border p-4">
        <h2 className="font-medium">Jurisdiction units</h2>
        <p className="text-sm">
          Search by name to find a unit id to paste into an approval below, or create a new one. A
          jurisdiction is optional (D-JurisdictionUnits, M4.1) — leave the approval field blank to
          keep the current flat-root behavior.
        </p>
        <form action="/admin/registrations" className="mt-2 flex gap-2">
          <input
            name="jurisdictionQuery"
            defaultValue={jurisdictionQuery}
            placeholder="Search jurisdiction units by name…"
            className="rounded border px-2 py-1 text-sm"
          />
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            Search
          </button>
        </form>
        {jurisdictionQuery && (
          <ul className="mt-3 flex flex-col gap-1 text-sm">
            {jurisdictionResults.length === 0 && <li>No matches.</li>}
            {jurisdictionResults.map((u) => (
              <li key={u.id} className="flex gap-2">
                <code className="rounded bg-gray-100 px-1">{u.id}</code>
                <span>{u.name}</span>
                {u.code && <span className="text-gray-500">({u.code})</span>}
              </li>
            ))}
          </ul>
        )}
        <form action={createJurisdiction} className="mt-3 flex flex-wrap gap-2">
          <input
            name="code"
            placeholder="New unit code"
            required
            className="rounded border px-2 py-1 text-sm"
          />
          <input
            name="name"
            placeholder="New unit name"
            required
            className="rounded border px-2 py-1 text-sm"
          />
          <input
            name="parentUnitId"
            placeholder="Parent unit id (blank = root)"
            className="rounded border px-2 py-1 text-sm"
          />
          <button type="submit" className="rounded border px-3 py-1 text-sm">
            Create jurisdiction unit
          </button>
        </form>
      </section>

      {requests.length === 0 && <p>No requests.</p>}
      <ul className="flex flex-col gap-4">
        {requests.map((r) => (
          <li key={r.id} className="rounded border p-4">
            <div className="flex items-baseline justify-between">
              <span className="font-medium">{r.congregationName}</span>
              <span className="text-sm">{r.status}</span>
            </div>
            <p className="text-sm">
              {r.locality ?? ""} {r.street ?? ""}
            </p>
            {r.status === "REJECTED" && r.rejectionReason && (
              <p className="text-sm">Reason: {r.rejectionReason}</p>
            )}
            {r.status === "PENDING" && (
              <div className="mt-3 flex flex-col gap-3">
                <form action={approve} className="flex gap-2">
                  <input type="hidden" name="id" value={r.id} />
                  <input
                    name="jurisdictionUnitId"
                    placeholder="Jurisdiction unit id (blank = root)"
                    className="rounded border px-2 py-1 text-sm"
                  />
                  <button type="submit" className="rounded border px-3 py-1 text-sm">
                    Approve
                  </button>
                </form>
                <form action={reject} className="flex gap-2">
                  <input type="hidden" name="id" value={r.id} />
                  <input
                    name="reason"
                    placeholder="Rejection reason"
                    required
                    className="rounded border px-2 py-1 text-sm"
                  />
                  <button type="submit" className="rounded border px-3 py-1 text-sm">
                    Reject
                  </button>
                </form>
              </div>
            )}
          </li>
        ))}
      </ul>
    </main>
  );
}
