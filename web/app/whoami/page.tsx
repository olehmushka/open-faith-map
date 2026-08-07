import { redirect } from "next/navigation";

import { auth, signOut } from "@/auth";
import { oikumenea } from "@/lib/oikumenea";

// The M1 exit-criterion proof artifact (docs/milestones.md): calls a real go-oikumenea endpoint
// with the logged-in user's forwarded Google ID token, proving the session layer's token passthrough
// works end-to-end, not just that login itself succeeds.
export default async function WhoamiPage() {
  const session = await auth();
  if (!session) {
    redirect("/login");
  }

  const client = await oikumenea();
  const who = await client.identityFederation.whoami();

  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col justify-center gap-4 px-6">
      <h1 className="text-2xl font-semibold">Who am I (via go-oikumenea)</h1>
      <p>
        Resolved by go-oikumenea&apos;s <code>identityFederation.whoami()</code> from the forwarded
        Google ID token — this is the M1 proof that login and token passthrough work end-to-end.
      </p>
      <pre className="overflow-x-auto rounded border p-4">{JSON.stringify(who, null, 2)}</pre>
      <form
        action={async () => {
          "use server";
          await signOut({ redirectTo: "/" });
        }}
      >
        <button type="submit" className="rounded border px-4 py-2">
          Sign out
        </button>
      </form>
    </main>
  );
}
