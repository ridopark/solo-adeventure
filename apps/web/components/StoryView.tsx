"use client";

import { useStory } from "@/hooks/useStory";
import { NarrativeBlock } from "./NarrativeBlock";
import { Illustration } from "./Illustration";
import { ChoiceButtons } from "./ChoiceButtons";
import { EndingCard } from "./EndingCard";
import { Skeleton } from "./Skeleton";

export function StoryView({ storyId }: { storyId: string }) {
  const { status, pages, current, error, choose, restart } = useStory(storyId);

  if (status === "hydrating" || !current) {
    return <Skeleton variant="full" />;
  }

  return (
    <article className="space-y-6">
      <Illustration src={current.imageUrl} alt={`Page ${current.index + 1}`} />
      <NarrativeBlock text={current.narrative} />
      {status === "choosing" && <Skeleton variant="next" />}
      {current.isEnding ? (
        <EndingCard endingType={current.endingType} onRestart={restart} />
      ) : (
        <ChoiceButtons
          choices={current.choices}
          disabled={status === "choosing"}
          onChoose={choose}
        />
      )}
      {error && <p className="text-sm text-rose-700">{error}</p>}
      <footer className="pt-4 text-xs text-stone-500">
        Page {current.index + 1} of {pages.length}
        {current.imageProvider ? ` — art via ${current.imageProvider}` : ""}
      </footer>
    </article>
  );
}
