import Link from "next/link";
import { AuthNav } from "./AuthNav";

export function SiteHeader() {
  return (
    <nav className="border-b border-[var(--border-cream)]">
      <div className="mx-auto max-w-5xl px-6 py-4 flex items-center justify-between">
        <Link href="/" className="font-serif text-xl tracking-tight">
          solo-adeventure
        </Link>
        <AuthNav />
      </div>
    </nav>
  );
}
