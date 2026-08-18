import { redirect } from "@/i18n/navigation";
import { whoami } from "@/lib/core";

// Gate 2 of D-SuperAdminFold's amendment (docs/architecture/decisions.md) — a
// requireInstanceAdmin() check in the super-admin route group's own layout, for COSMETIC gating
// only. The real gate is CoreSuperAdminService's whole-route-group RequireInstanceAdmin middleware
// (internal/authz/transport), already live since M10.7 — every call these pages make would 403 on
// its own even if this redirect were ever bypassed. This nests under admin/layout.tsx (which already
// ran the session-existence check) rather than duplicating it — admin/layout.tsx itself gets no role
// check, deliberately, since it's shared by every non-super-admin audience too.
export default async function SuperAdminLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const me = await whoami();
  if (!me.isInstanceAdmin) return redirect({ href: "/admin", locale });

  return <>{children}</>;
}
