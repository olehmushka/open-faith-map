import { redirect } from "next/navigation";

import { auth } from "@/auth";
import { approveRegistration, listRegistrations, rejectRegistration } from "@/lib/registration";

// Renders whatever listRegistrations returns for the caller — openfaithmap-api itself decides
// operator (all requests) vs. submitter (their own only) by asking go-oikumenea's PDP live
// (MyCapabilities), never a locally-cached role (D-Facade). A non-operator lands here and simply
// sees their own submissions; approve/reject actions still go through the real PDP check
// regardless of what this page renders (web-facade.md's "no client-side authorization").
export default async function RegistrationsPage() {
  const session = await auth();
  if (!session) redirect("/login");

  const { requests } = await listRegistrations();

  async function approve(formData: FormData) {
    "use server";
    await approveRegistration(String(formData.get("id")));
    redirect("/admin/registrations");
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
              <div className="mt-3 flex gap-3">
                <form action={approve}>
                  <input type="hidden" name="id" value={r.id} />
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
