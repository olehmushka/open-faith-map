import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "OpenFaithMap",
  description: "A free, open-source, Christian church-discovery-and-presence platform.",
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
