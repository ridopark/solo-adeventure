"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";

export function TopicInput() {
  const router = useRouter();
  const [topic, setTopic] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!topic.trim() || busy) return;
    setBusy(true);
    setError(null);
    try {
      const res = await api.startStory(topic.trim());
      sessionStorage.setItem(`solo-adv:seed:${res.storyId}`, JSON.stringify(res));
      router.push(`/story?id=${encodeURIComponent(res.storyId)}`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "failed to start";
      setError(extractError(msg));
      setBusy(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4 text-left">
      <label className="block space-y-2">
        <span className="uppercase tracking-[0.15em] text-xs text-[var(--stone-gray)]">
          Your topic
        </span>
        <input
          autoFocus
          value={topic}
          onChange={(e) => setTopic(e.target.value)}
          placeholder="a lighthouse keeper in 1912"
          className="w-full px-4 py-3 rounded-xl bg-white font-serif text-lg focus:outline-none transition"
          style={{
            border: "1px solid var(--border-warm)",
            color: "var(--near-black)",
          }}
          onFocus={(e) => (e.currentTarget.style.borderColor = "var(--terracotta)")}
          onBlur={(e) => (e.currentTarget.style.borderColor = "var(--border-warm)")}
        />
      </label>
      <button
        type="submit"
        disabled={!topic.trim() || busy}
        className="w-full px-5 py-3 rounded-xl font-medium text-base transition disabled:opacity-50 disabled:cursor-not-allowed"
        style={{
          background: "var(--terracotta)",
          color: "var(--ivory)",
          boxShadow: "var(--terracotta) 0px 0px 0px 1px",
        }}
        onMouseEnter={(e) => {
          if (!busy && topic.trim()) e.currentTarget.style.background = "var(--coral)";
        }}
        onMouseLeave={(e) => (e.currentTarget.style.background = "var(--terracotta)")}
      >
        {busy ? "Opening the book..." : "Begin"}
      </button>
      {error && (
        <p className="text-sm" style={{ color: "var(--crimson)" }}>
          {error}
        </p>
      )}
    </form>
  );
}

function extractError(msg: string): string {
  const m = msg.match(/"error":"([^"]+)"/);
  return m ? m[1] : msg;
}
