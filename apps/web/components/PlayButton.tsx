"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import { BACKEND_URL } from "@/lib/env";

type Status = "idle" | "loading" | "playing" | "paused" | "error";

function resolve(url: string) {
  return url.startsWith("http") ? url : `${BACKEND_URL}${url}`;
}

// edge-tts generates faster than realtime; empirically a 300-word page takes
// ~4-6s including network round-trip. Scale with narrative length, min 3s.
function estimateSeconds(chars: number): number {
  return Math.max(3, Math.round(chars * 0.003 + 2));
}

export function PlayButton({
  storyId,
  seq,
  initialAudioUrl,
  narrativeChars,
}: {
  storyId: string;
  seq: number;
  initialAudioUrl?: string;
  narrativeChars: number;
}) {
  const [status, setStatus] = useState<Status>(initialAudioUrl ? "idle" : "loading");
  const [audioUrl, setAudioUrl] = useState<string | undefined>(initialAudioUrl);
  const [etaSec, setEtaSec] = useState<number>(() => estimateSeconds(narrativeChars));
  const audioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(() => {
    setAudioUrl(initialAudioUrl);
    setStatus(initialAudioUrl ? "idle" : "loading");
    setEtaSec(estimateSeconds(narrativeChars));
    const el = audioRef.current;
    if (el) {
      el.pause();
      el.currentTime = 0;
    }
  }, [storyId, seq, initialAudioUrl, narrativeChars]);

  useEffect(() => {
    if (audioUrl) return;
    let cancelled = false;
    (async () => {
      try {
        const res = await api.speech(storyId, seq);
        if (cancelled) return;
        setAudioUrl(res.audioUrl);
        setStatus("idle");
      } catch {
        if (!cancelled) setStatus("error");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [storyId, seq, audioUrl]);

  useEffect(() => {
    if (status !== "loading") return;
    const t = setInterval(() => {
      setEtaSec((s) => Math.max(0, s - 1));
    }, 1000);
    return () => clearInterval(t);
  }, [status]);

  const play = useCallback(async () => {
    try {
      if (!audioUrl) return;
      const el = audioRef.current;
      if (!el) return;
      if (el.src !== resolve(audioUrl)) {
        el.src = resolve(audioUrl);
      }
      await el.play();
    } catch {
      setStatus("error");
    }
  }, [audioUrl]);

  const pause = useCallback(() => {
    audioRef.current?.pause();
    setStatus("paused");
  }, []);

  const isLoading = status === "loading";
  const isPlaying = status === "playing";
  const disabled = isLoading;
  const label = isLoading
    ? etaSec > 0
      ? `Generating narration... ~${etaSec}s`
      : "Almost ready..."
    : isPlaying
      ? "Pause"
      : status === "error"
        ? "Retry"
        : status === "paused"
          ? "Resume"
          : "Read aloud";

  return (
    <div className="flex items-center gap-2">
      <button
        type="button"
        onClick={isPlaying ? pause : play}
        disabled={disabled}
        aria-busy={isLoading}
        className={[
          "inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm transition",
          disabled
            ? "cursor-wait border-stone-200 bg-stone-100 text-stone-400"
            : "border-stone-300 bg-white/70 text-stone-700 hover:bg-white hover:text-stone-900",
        ].join(" ")}
      >
        {isLoading ? (
          <span
            aria-hidden
            className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-stone-300 border-t-stone-500"
          />
        ) : (
          <span aria-hidden className="text-xs leading-none">
            {isPlaying ? "\u275A\u275A" : "\u25B6"}
          </span>
        )}
        {label}
      </button>
      <audio
        ref={audioRef}
        src={audioUrl ? resolve(audioUrl) : undefined}
        preload="auto"
        onPlaying={() => setStatus("playing")}
        onWaiting={() => setStatus((s) => (s === "playing" ? "loading" : s))}
        onEnded={() => setStatus("idle")}
        onPause={() => setStatus((s) => (s === "playing" ? "paused" : s))}
      />
    </div>
  );
}
