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
      setError(err instanceof Error ? err.message : "failed to start");
      setBusy(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <label className="block space-y-2">
        <span className="text-sm text-stone-600">Your topic</span>
        <input
          autoFocus
          value={topic}
          onChange={(e) => setTopic(e.target.value)}
          placeholder="a lighthouse keeper in 1912"
          className="w-full px-4 py-3 rounded-sm border border-stone-400 bg-[#fbf6ea] focus:outline-none focus:border-stone-700"
        />
      </label>
      <button
        type="submit"
        disabled={!topic.trim() || busy}
        className="w-full px-5 py-3 border border-stone-700 rounded-sm bg-stone-900 text-stone-50 hover:bg-stone-800 disabled:opacity-40 transition"
      >
        {busy ? "Opening the book..." : "Begin"}
      </button>
      {error && <p className="text-sm text-rose-700">{error}</p>}
    </form>
  );
}
