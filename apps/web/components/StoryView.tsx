"use client";

import { useStory } from "@/hooks/useStory";
import { NarrativeBlock } from "./NarrativeBlock";
import { Illustration } from "./Illustration";
import { ChoiceButtons } from "./ChoiceButtons";
import { EndingCard } from "./EndingCard";
import { Skeleton } from "./Skeleton";
import { SignInPrompt } from "./SignInPrompt";
import { PlayButton } from "./PlayButton";

export function StoryView({ storyId }: { storyId: string }) {
  const { status, pages, current, error, choose, restart } = useStory(storyId);

  if (status === "hydrating" || !current) {
    return <Skeleton variant="full" />;
  }

  return (
    <article className="space-y-6">
      <Illustration src={current.imageUrl} alt={`Page ${current.index + 1}`} seq={current.index} />
      <NarrativeBlock text={current.narrative} />
      <PlayButton
        storyId={storyId}
        seq={current.index}
        initialAudioUrl={current.audioUrl}
        narrativeChars={current.narrative.length}
      />
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
      {error && <p className="text-sm" style={{ color: "var(--crimson)" }}>{error}</p>}
      <footer className="pt-4 text-xs" style={{ color: "var(--stone-gray)" }}>
        Page {current.index + 1} of {pages.length}
        {current.imageProvider ? ` -- art via ${current.imageProvider}` : ""}
      </footer>
    </article>
  );
}
