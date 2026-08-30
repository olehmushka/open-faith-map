"use client";

import { useTranslations } from "next-intl";
import {
  ArchiveRestore,
  Blocks,
  BookOpen,
  Building2,
  Flag,
  History,
  KeyRound,
  LayoutTemplate,
  MapPinned,
  Search,
  ShieldCheck,
  UserCog,
  UserPlus,
} from "lucide-react";

import { Link, usePathname } from "@/i18n/navigation";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar";

type NavItem = {
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  labelKey: string;
  children?: { href: string; labelKey: string }[];
};

export const NAV: NavItem[] = [
  {
    href: "/admin/congregation-import",
    icon: MapPinned,
    labelKey: "congregationImport",
    children: [{ href: "/admin/congregation-import/aliases", labelKey: "aliases" }],
  },
  {
    href: "/admin/moderation",
    icon: Flag,
    labelKey: "moderation",
    children: [{ href: "/admin/moderation/appeals", labelKey: "appeals" }],
  },
  {
    href: "/admin/registrations",
    icon: UserPlus,
    labelKey: "registrations",
    children: [{ href: "/admin/registrations/reparent", labelKey: "reparent" }],
  },
  {
    href: "/admin/vouching",
    icon: ShieldCheck,
    labelKey: "vouching",
    children: [{ href: "/admin/vouching/new", labelKey: "newVouch" }],
  },
  {
    href: "/admin/sites",
    icon: ArchiveRestore,
    labelKey: "sites",
  },
  // M14.13: content.catalog.manage-gated server-side (platform-moderator, D-SitePatterns) — same
  // "no local isModerator gate, shown unconditionally" discipline /admin/moderation above already
  // follows. A non-moderator sees these links but is denied server-side, not hidden from the nav.
  {
    href: "/admin/block-types",
    icon: Blocks,
    labelKey: "blockTypes",
  },
  {
    href: "/admin/patterns",
    icon: LayoutTemplate,
    labelKey: "patterns",
  },
];

// M10.8's four super-admin screens (D-SuperAdminFold) — a separate group, not merged into NAV, so
// the sidebar visually distinguishes "every admin audience" from "instance-admin only" even though
// (per D-SuperAdminFold's amendment) this list is NOT itself a security boundary: every item here is
// shown unconditionally, exactly like every other NAV entry — a congregation-admin who clicks
// "People" gets redirected by the (super-admin) route group's own layout, the same way a non-operator
// who clicks "Registrations" today still sees the link but gets denied server-side, not a hidden one.
export const SUPER_ADMIN_NAV: NavItem[] = [
  { href: "/admin/people", icon: UserCog, labelKey: "people" },
  { href: "/admin/role-grants", icon: KeyRound, labelKey: "roleGrants" },
  { href: "/admin/units", icon: Building2, labelKey: "units" },
  { href: "/admin/taxa", icon: BookOpen, labelKey: "taxa" },
  { href: "/admin/audit-log", icon: History, labelKey: "auditLog" },
  { href: "/admin/explain-access", icon: Search, labelKey: "explainAccess" },
];

function NavItems({
  items,
  pathname,
  t,
}: {
  items: NavItem[];
  pathname: string;
  t: (key: string) => string;
}) {
  return (
    <>
      {items.map((item) => {
        const active = pathname === item.href || pathname.startsWith(item.href + "/");
        return (
          <SidebarMenuItem key={item.href}>
            <SidebarMenuButton asChild isActive={active} tooltip={t(item.labelKey)}>
              <Link href={item.href}>
                <item.icon className="size-4" />
                <span>{t(item.labelKey)}</span>
              </Link>
            </SidebarMenuButton>
            {item.children && (
              <SidebarMenuSub>
                {item.children.map((child) => (
                  <SidebarMenuSubItem key={child.href}>
                    <SidebarMenuSubButton asChild isActive={pathname === child.href}>
                      <Link href={child.href}>{t(child.labelKey)}</Link>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                ))}
              </SidebarMenuSub>
            )}
          </SidebarMenuItem>
        );
      })}
    </>
  );
}

export function AdminSidebar() {
  const pathname = usePathname();
  const t = useTranslations("AdminShell");

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <div className="flex items-center gap-2 px-2 py-1 text-sm font-semibold">
          {t("brand")}
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>{t("sections")}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <NavItems items={NAV} pathname={pathname} t={t} />
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarGroup>
          <SidebarGroupLabel>{t("superAdminSection")}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <NavItems items={SUPER_ADMIN_NAV} pathname={pathname} t={t} />
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  );
}
