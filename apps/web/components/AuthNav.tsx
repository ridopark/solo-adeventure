"use client";

import Link from "next/link";
import { useAuth } from "@/hooks/useAuth";
import { api } from "@/lib/api";

export function AuthNav() {
  const { user, loading, logout } = useAuth();

  if (loading) {
    return <span className="text-sm text-[var(--stone-gray)]">&nbsp;</span>;
  }

  if (!user) {
    return (
      <a
        href={api.loginURL(typeof window !== "undefined" ? window.location.href : undefined)}
        className="text-sm px-3 py-1.5 rounded-lg transition"
        style={{ background: "var(--warm-sand)", color: "var(--charcoal-warm)", boxShadow: "var(--ring-warm) 0px 0px 0px 1px" }}
      >
        Sign in with Google
      </a>
    );
  }

  return (
    <div className="flex items-center gap-4 text-sm">
      <Link href="/my-stories" className="text-[var(--olive-gray)] hover:text-[var(--near-black)] transition">
        My stories
      </Link>
      <div className="flex items-center gap-2">
        {user.avatarUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={user.avatarUrl} alt="" className="w-7 h-7 rounded-full" />
        ) : (
          <span className="w-7 h-7 rounded-full bg-[var(--warm-sand)] flex items-center justify-center text-xs">
            {user.name?.[0]?.toUpperCase() ?? "?"}
          </span>
        )}
        <span className="text-[var(--olive-gray)] hidden sm:inline">{user.name || user.email}</span>
      </div>
      <button
        onClick={logout}
        className="text-[var(--stone-gray)] hover:text-[var(--near-black)] transition"
      >
        Sign out
      </button>
    </div>
  );
}
