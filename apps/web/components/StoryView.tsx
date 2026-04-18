"use client";

import { useEffect, useState } from "react";
import { useStory } from "@/hooks/useStory";
import { api } from "@/lib/api";
import { NarrativeBlock } from "./NarrativeBlock";
import { Illustration } from "./Illustration";
import { ParallaxIllustration } from "./ParallaxIllustration";
import { ChoiceButtons } from "./ChoiceButtons";
import { EndingCard } from "./EndingCard";
import { Skeleton } from "./Skeleton";
import { SignInPrompt } from "./SignInPrompt";
import { PlayButton } from "./PlayButton";
import { ShareButton } from "./ShareButton";

export function StoryView({ storyId, startAt }: { storyId: string; startAt?: number }) {
  const {
    status,
    pages,
    current,
    cursor,
    atLatest,
    canGoPrev,
    canGoNext,
    error,
    choose,
    goPrev,
    goNext,
    goLatest,
    restart,
  } = useStory(storyId, { startAt });
  const [depthUrl, setDepthUrl] = useState<string | undefined>(current?.depthUrl);
  const [paused, setPaused] = useState(false);
  const [showOriginal, setShowOriginal] = useState(false);

  useEffect(() => {
    setPaused(false);
    setShowOriginal(false);
  }, [current?.index]);

  useEffect(() => {
    setDepthUrl(current?.depthUrl);
    if (!current || current.depthUrl || !current.imageUrl) {
      if (current?.depthUrl) console.log(`[story] page seq=${current.index} already has depth`);
      return;
    }
    console.log(`[story] requesting depth for seq=${current.index}`);
    const t0 = performance.now();
    let cancelled = false;
    api
      .depth(storyId, current.index)
      .then((res) => {
        if (cancelled) return;
        console.log(`[story] depth ready in ${Math.round(performance.now() - t0)}ms: ${res.depthUrl}`);
        setDepthUrl(res.depthUrl);
      })
      .catch((e) => {
        console.warn(`[story] depth request failed after ${Math.round(performance.now() - t0)}ms`, e);
      });
    return () => {
      cancelled = true;
    };
  }, [storyId, current?.index, current?.imageUrl, current?.depthUrl]);

  if (status === "hydrating" || !current) {
    return <Skeleton variant="full" />;
  }

  const hasParallax = Boolean(current.imageUrl && depthUrl);
  const illustration =
    hasParallax && !showOriginal ? (
      <ParallaxIllustration
        imageSrc={current.imageUrl!}
        depthSrc={depthUrl!}
        alt={`Page ${current.index + 1}`}
        seq={current.index}
        paused={paused}
      />
    ) : (
      <Illustration
        src={current.imageUrl}
        alt={`Page ${current.index + 1}`}
        seq={current.index}
        still={showOriginal}
      />
    );

  return (
    <article className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] lg:items-start lg:gap-10">
      <div className="lg:sticky lg:top-8">{illustration}</div>
      <div className="space-y-6">
        <NarrativeBlock text={current.narrative} />
        <div className="flex flex-wrap items-center gap-2">
          <PlayButton
            storyId={storyId}
            seq={current.index}
            initialAudioUrl={current.audioUrl}
            narrativeChars={current.narrative.length}
          />
          <ShareButton storyId={storyId} />
          {hasParallax && (
            <>
              <button
                type="button"
                onClick={() => setShowOriginal((v) => !v)}
                aria-pressed={showOriginal}
                className="inline-flex items-center gap-2 rounded-full border border-stone-300 bg-white/70 px-3 py-1.5 text-sm text-stone-700 transition hover:bg-white hover:text-stone-900"
              >
                {showOriginal ? "Show 3D" : "Show original"}
              </button>
              {!showOriginal && (
                <button
                  type="button"
                  onClick={() => setPaused((v) => !v)}
                  aria-pressed={paused}
                  className="inline-flex items-center gap-2 rounded-full border border-stone-300 bg-white/70 px-3 py-1.5 text-sm text-stone-700 transition hover:bg-white hover:text-stone-900"
                >
                  {paused ? "Resume motion" : "Pause motion"}
                </button>
              )}
            </>
          )}
        </div>
        {atLatest ? (
          <>
            {status === "choosing" && <Skeleton variant="next" />}
            {status === "needs_auth" && <SignInPrompt />}
            {current.isEnding ? (
              <EndingCard endingType={current.endingType} onRestart={restart} />
            ) : (
              <ChoiceButtons
                choices={current.choices}
                disabled={status === "choosing" || status === "needs_auth"}
                onChoose={choose}
              />
            )}
          </>
        ) : (
          <div className="rounded-md border border-stone-300 bg-stone-50 px-4 py-3 text-sm text-stone-600">
            You're reading an earlier page.{" "}
            <button
              type="button"
              onClick={goLatest}
              className="underline underline-offset-2 hover:text-stone-900"
            >
              Jump to the current page &rarr;
            </button>
          </div>
        )}
        {error && <p className="text-sm" style={{ color: "var(--crimson)" }}>{error}</p>}
        <footer className="flex items-center justify-between pt-4 text-xs" style={{ color: "var(--stone-gray)" }}>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={goPrev}
              disabled={!canGoPrev}
              className="rounded-full border border-stone-300 px-2 py-1 disabled:opacity-30 disabled:cursor-not-allowed enabled:hover:bg-white"
              aria-label="Previous page"
            >
              &larr; Prev
            </button>
            <button
              type="button"
              onClick={goNext}
              disabled={!canGoNext}
              className="rounded-full border border-stone-300 px-2 py-1 disabled:opacity-30 disabled:cursor-not-allowed enabled:hover:bg-white"
              aria-label="Next page"
            >
              Next &rarr;
            </button>
          </div>
          <div>
            Page {cursor + 1} of {pages.length}
            {current.imageProvider ? ` -- art via ${current.imageProvider}` : ""}
          </div>
        </footer>
      </div>
    </article>
  );
}
