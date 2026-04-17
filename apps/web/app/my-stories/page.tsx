"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import type { Story } from "@/lib/types";
import { AuthNav } from "@/components/AuthNav";

type Status = "loading" | "ready" | "unauthenticated" | "error";

export default function MyStoriesPage() {
  const [status, setStatus] = useState<Status>("loading");
  const [stories, setStories] = useState<Story[]>([]);

  useEffect(() => {
    api
      .myStories()
      .then((res) => {
        setStories(res.stories ?? []);
        setStatus("ready");
      })
      .catch((err: unknown) => {
        const s = (err as { status?: number }).status;
        setStatus(s === 401 ? "unauthenticated" : "error");
      });
  }, []);

  return (
    <div>
      <nav className="border-b border-[var(--border-cream)]">
        <div className="mx-auto max-w-5xl px-6 py-4 flex items-center justify-between">
          <Link href="/" className="font-serif text-xl tracking-tight">
            solo-adeventure
          </Link>
          <AuthNav />
        </div>
      </nav>

      <main className="mx-auto max-w-3xl px-6 py-16">
        <h1 className="font-serif text-4xl leading-[1.10] tracking-tight mb-8 text-[var(--near-black)]">
          My stories
        </h1>

        {status === "loading" && <p className="text-[var(--stone-gray)]">Loading...</p>}

        {status === "unauthenticated" && (
          <p className="text-[var(--olive-gray)]">
            Sign in from the nav to see your saved stories.
          </p>
        )}

        {status === "error" && (
          <p style={{ color: "var(--crimson)" }}>Something went wrong. Try refreshing.</p>
        )}

        {status === "ready" && stories.length === 0 && (
          <p className="text-[var(--olive-gray)]">
            No stories yet. <Link href="/" className="underline">Start one</Link>.
          </p>
        )}

        {status === "ready" && stories.length > 0 && (
          <ul className="space-y-4">
            {stories.map((s) => (
              <li key={s.storyId}>
                <Link
                  href={`/story?id=${encodeURIComponent(s.storyId)}`}
                  className="block rounded-2xl p-6 transition"
                  style={{
                    background: "var(--ivory)",
                    border: "1px solid var(--border-cream)",
                  }}
                >
                  <h2 className="font-serif text-xl text-[var(--near-black)] mb-2">
                    {s.topic}
                  </h2>
                  <p className="text-sm text-[var(--stone-gray)]">
                    {new Date(s.updatedAt).toLocaleString()}
                  </p>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  );
}
