import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Recurso Starter",
  description: "A minimal Next.js SaaS starter powered by the Recurso billing engine",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <nav className="topnav">
          <Link href="/" className="brand">
            Recurso Starter
          </Link>
          <Link href="/pricing">Pricing</Link>
          <Link href="/account">Account</Link>
          <Link href="/feature">Reports</Link>
        </nav>
        <main className="container">{children}</main>
      </body>
    </html>
  );
}
