import { notFound } from "next/navigation";

// Sibling to admin/ and the (chrome) route group — Next.js always prefers a matching static
// segment (admin, login, register, whoami, my-congregation) over this catch-all, so it only
// matches genuinely unknown top-level paths and renders not-found.tsx below.
export default function LocaleCatchAll() {
  notFound();
}
