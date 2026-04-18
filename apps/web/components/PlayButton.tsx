"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import { BACKEND_URL } from "@/lib/env";

type Status = "idle" | "loading" | "playing" | "paused" | "error";

function resolve(url: string) {
  return url.startsWith("http") ? url : `${BACKEND_URL}${url}`;
}

export function PlayButton({
  storyId,
  seq,
  initialAudioUrl,
}: {
  storyId: string;
  seq: number;
  initialAudioUrl?: string;
}) {
  const [status, setStatus] = useState<Status>("idle");
  const [audioUrl, setAudioUrl] = useState<string | undefined>(initialAudioUrl);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(() => {
    setAudioUrl(initialAudioUrl);
    setStatus("idle");
    const el = audioRef.current;
    if (el) {
      el.pause();
      el.currentTime = 0;
    }
  }, [storyId, seq, initialAudioUrl]);

  const play = useCallback(async () => {
    try {
      let url = audioUrl;
      if (!url) {
        setStatus("loading");
        const res = await api.speech(storyId, seq);
        url = res.audioUrl;
        setAudioUrl(url);
      }
      const el = audioRef.current;
      if (!el) return;
      if (el.src !== resolve(url)) {
        el.src = resolve(url);
      }
      await el.play();
      setStatus("playing");
    } catch {
      setStatus("error");
    }
  }, [audioUrl, storyId, seq]);

  const pause = useCallback(() => {
    audioRef.current?.pause();
    setStatus("paused");
  }, []);

  const label =
    status === "loading"
      ? "Loading..."
      : status === "playing"
        ? "Pause"
        : status === "error"
          ? "Retry"
          : "Read aloud";

  return (
    <div className="flex items-center gap-2">
      <button
        type="button"
        onClick={status === "playing" ? pause : play}
        disabled={status === "loading"}
        className="inline-flex items-center gap-2 rounded-full border border-stone-300 bg-white/70 px-3 py-1.5 text-sm text-stone-700 transition hover:bg-white hover:text-stone-900 disabled:opacity-50"
      >
        <span aria-hidden>{status === "playing" ? "\u275A\u275A" : "\u25B6"}</span>
        {label}
      </button>
      <audio
        ref={audioRef}
        preload="none"
        onEnded={() => setStatus("idle")}
        onPause={() => setStatus((s) => (s === "playing" ? "paused" : s))}
      />
    </div>
  );
}
