"use client";

import { api } from "@/lib/api";

export function SignInPrompt() {
  const returnTo = typeof window !== "undefined" ? window.location.href : undefined;
  return (
    <aside
      className="rounded-2xl p-6 space-y-4"
      style={{
        background: "var(--ivory)",
        border: "1px solid var(--border-warm)",
      }}
    >
      <div>
        <p className="uppercase tracking-[0.15em] text-xs text-[var(--stone-gray)] mb-2">
          Sign in to continue
        </p>
        <h3 className="font-serif text-2xl text-[var(--near-black)] mb-2">
          Save this tale to your shelf.
        </h3>
        <p className="text-base leading-[1.60] text-[var(--olive-gray)]">
          Reading past page one needs a signed-in reader so your choices and the
          book's memory can be saved and resumed from any device.
        </p>
      </div>
      <a
        href={api.loginURL(returnTo)}
        className="inline-block px-5 py-3 rounded-xl font-medium transition"
        style={{
          background: "var(--terracotta)",
          color: "var(--ivory)",
          boxShadow: "var(--terracotta) 0px 0px 0px 1px",
        }}
      >
        Sign in with Google
      </a>
    </aside>
  );
}
