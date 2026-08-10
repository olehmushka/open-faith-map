import Link from "next/link";
import { redirect } from "next/navigation";

import { auth } from "@/auth";
import { getSite, listDocuments } from "@/lib/content";

export default async function DocumentsPage({ params }: { params: Promise<{ unitId: string }> }) {
  const session = await auth();
  if (!session) redirect("/login");

  const { unitId } = await params;
  const site = await getSite(unitId).catch(() => null);
  if (!site) redirect(`/admin/sites/${unitId}`);

  const documents = await listDocuments(site.id);

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col gap-6 px-6 py-12">
      <h1 className="text-2xl font-semibold">Pages</h1>
      <Link href={`/admin/sites/${unitId}/documents/new`} className="underline">
        New page
      </Link>

      {documents.length === 0 && <p>No pages yet.</p>}
      <ul className="flex flex-col gap-3">
        {documents.map((d) => (
          <li key={d.id} className="rounded border p-4">
            <div className="flex items-baseline justify-between">
              <Link href={`/admin/sites/${unitId}/documents/${d.id}`} className="font-medium underline">
                {d.slug}
              </Link>
              <span className={`text-sm ${d.state === "DRAFT" ? "font-semibold" : ""}`}>
                {d.state === "DRAFT" ? "DRAFT (not public)" : d.state}
              </span>
            </div>
            <p className="text-sm">
              {d.locale} · {d.kind}
            </p>
          </li>
        ))}
      </ul>
    </main>
  );
}
