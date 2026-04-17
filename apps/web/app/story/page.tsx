"use client";

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { StoryView } from "@/components/StoryView";

function StoryScreen() {
  const params = useSearchParams();
  const id = params.get("id");
  if (!id) {
    return (
      <p className="text-rose-700">
        Missing story id. Start a new adventure from the home page.
      </p>
    );
  }
  return <StoryView storyId={id} />;
}

export default function Page() {
  return (
    <Suspense fallback={<p className="text-stone-500">Loading...</p>}>
      <StoryScreen />
    </Suspense>
  );
}
