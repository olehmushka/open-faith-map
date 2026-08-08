import { signIn } from "@/auth";

export default function LoginPage() {
  return (
    <main className="mx-auto flex min-h-screen max-w-sm flex-col justify-center gap-4 px-6">
      <h1 className="text-xl font-semibold">Sign in</h1>
      <form
        action={async () => {
          "use server";
          await signIn("google", { redirectTo: "/whoami" });
        }}
      >
        <button type="submit" className="rounded border px-4 py-2">
          Sign in with Google
        </button>
      </form>
    </main>
  );
}
