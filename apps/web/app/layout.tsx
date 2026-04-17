import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "solo-adeventure",
  description: "Your own choose-your-own-adventure, powered by Claude.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-[#f5efe1] text-stone-900 font-serif antialiased">
        <main className="mx-auto max-w-2xl px-6 py-10">{children}</main>
      </body>
    </html>
  );
}
