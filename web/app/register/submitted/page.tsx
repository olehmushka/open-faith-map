import Link from "next/link";

import { getRegistration } from "@/lib/registration";

export default async function RegisterSubmittedPage({
  searchParams,
}: {
  searchParams: Promise<{ id?: string }>;
}) {
  const { id } = await searchParams;
  const req = id ? await getRegistration(id).catch(() => null) : null;

  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-4 px-6 py-12">
      <h1 className="text-2xl font-semibold">Request submitted</h1>
      {req ? (
        <p>
          Your registration for <strong>{req.congregationName}</strong> is pending review. Status:{" "}
          {req.status}.
        </p>
      ) : (
        <p>Your registration has been submitted and is pending review.</p>
      )}
      <Link href="/" className="underline">
        Back to home
      </Link>
    </main>
  );
}
