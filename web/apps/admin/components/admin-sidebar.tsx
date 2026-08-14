"use client";

import { useTranslations } from "next-intl";
import {
  ArchiveRestore,
  Flag,
  MapPinned,
  ShieldCheck,
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
];

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
              {NAV.map((item) => {
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
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  );
}
