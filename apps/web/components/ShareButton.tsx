"use client";

import { useState } from "react";

type Status = "idle" | "copied" | "error";

export function ShareButton({ storyId }: { storyId: string }) {
  const [status, setStatus] = useState<Status>("idle");

  const handleShare = async () => {
    const url =
      typeof window !== "undefined"
        ? `${window.location.origin}/story?id=${storyId}`
        : `/story?id=${storyId}`;
    const shareData = {
      title: "A solo-adeventure story",
      text: "Come read this choose-your-own-adventure with me.",
      url,
    };

    if (typeof navigator !== "undefined" && typeof navigator.share === "function") {
      try {
        await navigator.share(shareData);
        return;
      } catch (e) {
        if ((e as Error).name === "AbortError") return;
        // fall through to clipboard if share sheet itself errored
      }
    }

    try {
      await navigator.clipboard.writeText(url);
      setStatus("copied");
    } catch {
      setStatus("error");
    }
    setTimeout(() => setStatus("idle"), 2500);
  };

  const label =
    status === "copied" ? "Link copied" : status === "error" ? "Copy failed" : "Share";

  return (
    <button
      type="button"
      onClick={handleShare}
      aria-label="Share this story"
      className="inline-flex items-center gap-2 rounded-full border border-stone-300 bg-white/70 px-3 py-1.5 text-sm text-stone-700 transition hover:bg-white hover:text-stone-900"
    >
      <span aria-hidden className="text-xs leading-none">
        {"\u2197"}
      </span>
      {label}
    </button>
  );
}
