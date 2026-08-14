import { auth } from "@/auth";
import { redirect } from "@/i18n/navigation";
import { searchJurisdictionUnits } from "@/lib/jurisdiction";
import { AdminSidebar } from "@/components/admin-sidebar";
import { AdminTopbar } from "@/components/admin-topbar";
import { CommandPalette } from "@/components/command-palette";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "@/components/ui/sonner";

// Centralizes the session-existence check that every admin page used to duplicate
// (`const session = await auth(); if (!session) redirect(...)`). This only removes duplicated
// "is anyone logged in" boilerplate — it adds no role/permission gate. Every mutation still relies
// exclusively on go-oikumenea's live PDP (D-Facade, see docs/modules/web-admin.md); a non-operator's
// approve/reject/etc. calls still come back Forbidden from openfaithmap-api itself, same as before.
export default async function AdminLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const session = await auth();
  if (!session) return redirect({ href: "/login", locale });

  async function searchSites(query: string) {
    "use server";
    return searchJurisdictionUnits(query);
  }

  return (
    <TooltipProvider>
      <SidebarProvider>
        <AdminSidebar />
        <SidebarInset>
          <AdminTopbar user={session.user} locale={locale} />
          <main className="flex flex-1 flex-col gap-6 p-6">{children}</main>
        </SidebarInset>
        <CommandPalette onSearchSite={searchSites} />
        <Toaster />
      </SidebarProvider>
    </TooltipProvider>
  );
}
