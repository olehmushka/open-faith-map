import { getTranslations } from "next-intl/server";
import { LogOut } from "lucide-react";
import type { DefaultSession } from "next-auth";

import { signOut } from "@/auth";
import { LocaleSwitcher } from "@/app/[locale]/locale-switcher";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { CommandPaletteButton } from "@/components/command-palette-button";

function initials(name?: string | null) {
  if (!name) return "?";
  return name
    .split(" ")
    .map((part) => part[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

export async function AdminTopbar({
  user,
  locale,
}: {
  user: DefaultSession["user"];
  locale: string;
}) {
  const t = await getTranslations("AdminShell");

  return (
    <header className="flex items-center justify-between gap-3 border-b px-4 py-2">
      <div className="flex items-center gap-2">
        <SidebarTrigger />
      </div>
      <div className="flex items-center gap-2">
        <CommandPaletteButton />
        <LocaleSwitcher />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="rounded-full">
              <Avatar className="size-7">
                <AvatarImage src={user?.image ?? undefined} alt="" />
                <AvatarFallback>{initials(user?.name)}</AvatarFallback>
              </Avatar>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel className="font-normal">
              <div className="flex flex-col gap-0.5">
                <span className="text-sm font-medium">{user?.name ?? user?.email}</span>
                {user?.name && <span className="text-xs text-muted-foreground">{user.email}</span>}
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <form
              action={async () => {
                "use server";
                await signOut({ redirectTo: `/${locale}` });
              }}
            >
              <DropdownMenuItem asChild>
                <button type="submit" className="w-full">
                  <LogOut />
                  {t("signOut")}
                </button>
              </DropdownMenuItem>
            </form>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
