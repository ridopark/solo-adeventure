import type { Metadata } from "next";
import { VisitPing } from "@/components/VisitPing";
import "./globals.css";

export const metadata: Metadata = {
  title: "solo-adeventure -- a choose-your-own-adventure gamebook",
  description:
    "Pick a topic. Claude writes each page of your adventure as you turn it; FLUX illustrates. No two tales alike.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen antialiased" style={{ background: "var(--parchment)", color: "var(--near-black)" }}>
        <VisitPing />
        {children}
      </body>
    </html>
  );
}
