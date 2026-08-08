import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "OpenFaithMap Admin",
  description:
    "OpenFaithMap's admin/moderator console — registration wizard, operator-approval console, congregation-admin console, moderator console. See docs/modules/web-admin.md.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
