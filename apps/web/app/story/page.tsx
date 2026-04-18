"use client";

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { StoryView } from "@/components/StoryView";

function StoryScreen() {
  const params = useSearchParams();
  const id = params.get("id");
  const from = params.get("from");
  if (!id) {
    return (
      <p style={{ color: "var(--crimson)" }}>
        Missing story id. Start a new adventure from the home page.
      </p>
    );
  }
  return <StoryView storyId={id} startAt={from === "start" ? 0 : undefined} />;
}

export default function Page() {
  return (
    <main className="mx-auto max-w-2xl lg:max-w-5xl px-6 py-10">
      <Suspense fallback={<p style={{ color: "var(--stone-gray)" }}>Loading...</p>}>
        <StoryScreen />
      </Suspense>
    </main>
  );
}
