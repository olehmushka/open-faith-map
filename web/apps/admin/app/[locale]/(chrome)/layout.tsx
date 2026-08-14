import { LocaleSwitcher } from "../locale-switcher";

// Shared chrome for every non-admin route (login, register, whoami, my-congregation) — the plain
// header + locale switcher that used to live in the root layout. Admin routes get their own shell
// (sidebar + topbar) from app/[locale]/admin/layout.tsx instead.
export default function ChromeLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <header className="flex justify-end border-b px-6 py-2">
        <LocaleSwitcher />
      </header>
      {children}
    </>
  );
}
