import { redirect } from "next/navigation";

import { auth } from "@/auth";
import { createJurisdictionUnit, searchJurisdictionUnits } from "@/lib/jurisdiction";
import { getReparentStatus, listRegistrations, reparentRegistration } from "@/lib/registration";

// Re-parenting an already-APPROVED congregation's unit onto a different jurisdiction (M4.1,
// D-JurisdictionUnits) — separate from the approval flow (page.tsx) since it acts on a congregation
// that's already live, one at a time, live-verified before batching per the milestone's own
// "highest-risk item" framing. Same operator-only gate as approve/list — openfaithmap-api's own PDP
// check decides for real, this page renders whatever listRegistrations already returns.
export default async function ReparentPage({
  searchParams,
}: {
  searchParams: Promise<{ jurisdictionQuery?: string }>;
}) {
  const session = await auth();
  if (!session) redirect("/login");

  const { requests } = await listRegistrations("APPROVED");
  const { jurisdictionQuery } = await searchParams;
  const jurisdictionResults = jurisdictionQuery ? await searchJurisdictionUnits(jurisdictionQuery) : [];

  const jobs = await Promise.all(
    requests.map(async (r) => ({ requestId: r.id, job: await getReparentStatus(r.id) })),
  );
  const jobByRequestId = new Map(jobs.map((j) => [j.requestId, j.job]));

  async function createJurisdiction(formData: FormData) {
    "use server";
    const parentUnitId = String(formData.get("parentUnitId") ?? "").trim() || undefined;
    const code = String(formData.get("code") ?? "").trim();
    const name = String(formData.get("name") ?? "").trim();
    if (!code || !name) return;
    const rootUnitId = process.env.REGISTRATION_ROOT_UNIT_ID;
    if (!rootUnitId) throw new Error("REGISTRATION_ROOT_UNIT_ID is not set.");
    const unit = await createJurisdictionUnit(parentUnitId ?? rootUnitId, code, name);
    redirect(`/admin/registrations/reparent?jurisdictionQuery=${encodeURIComponent(unit.name)}`);
  }

  async function reparent(formData: FormData) {
    "use server";
    const id = String(formData.get("id"));
    const newParentUnitId = String(formData.get("newParentUnitId") ?? "").trim();
    if (!newParentUnitId) return;
    await reparentRegistration(id, newParentUnitId);
    redirect("/admin/registrations/reparent");
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">Re-parent congregations</h1>
      <p className="text-sm">
        Move an already-approved congregation onto a different jurisdiction unit. This is a
        resumable, two-step move on go-oikumenea's side (add the new parent, then remove the old
        one) — re-submitting the same target for a congregation with a job already in progress
        resumes it rather than starting over.
      </p>

      <section className="rounded border p-4">
        <h2 className="font-medium">Jurisdiction units</h2>
        <form action="/admin/registrations/reparent" className="mt-2 flex gap-2">
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
          <input name="code" placeholder="New unit code" required className="rounded border px-2 py-1 text-sm" />
          <input name="name" placeholder="New unit name" required className="rounded border px-2 py-1 text-sm" />
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

      {requests.length === 0 && <p>No approved congregations.</p>}
      <ul className="flex flex-col gap-4">
        {requests.map((r) => {
          const job = jobByRequestId.get(r.id) ?? null;
          return (
            <li key={r.id} className="rounded border p-4">
              <div className="flex items-baseline justify-between">
                <span className="font-medium">{r.congregationName}</span>
                <span className="text-sm text-gray-500">unit {r.createdUnitId}</span>
              </div>
              <p className="text-sm">
                current jurisdiction: {r.jurisdictionUnitId ?? "(none — direct child of root)"}
              </p>
              {job && (
                <p className="text-sm">
                  last move: {job.oldParentUnitId} → {job.newParentUnitId} — {job.status}
                  {job.error && <span className="text-red-600"> ({job.error})</span>}
                </p>
              )}
              <form action={reparent} className="mt-3 flex gap-2">
                <input type="hidden" name="id" value={r.id} />
                <input
                  name="newParentUnitId"
                  placeholder="New parent unit id"
                  required
                  className="rounded border px-2 py-1 text-sm"
                />
                <button type="submit" className="rounded border px-3 py-1 text-sm">
                  {job && job.status !== "VERIFIED" && job.status !== "FAILED" ? "Resume move" : "Re-parent"}
                </button>
              </form>
            </li>
          );
        })}
      </ul>
    </main>
  );
}
