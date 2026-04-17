"use client";

import type { EndingType } from "@/lib/types";

const tint: Record<EndingType, string> = {
  victory: "bg-emerald-50 border-emerald-600 text-emerald-900",
  defeat: "bg-rose-50 border-rose-700 text-rose-900",
  twist: "bg-indigo-50 border-indigo-600 text-indigo-900",
};

export function EndingCard({
  endingType,
  onRestart,
}: {
  endingType?: EndingType;
  onRestart: () => void;
}) {
  const cls = endingType ? tint[endingType] : "bg-stone-100 border-stone-500 text-stone-900";
  return (
    <aside className={`border rounded-sm px-5 py-4 space-y-3 ${cls}`}>
      <p className="uppercase tracking-wide text-xs">
        {endingType ? `Ending: ${endingType}` : "The end"}
      </p>
      <div className="flex gap-2">
        <button
          onClick={onRestart}
          className="px-4 py-2 border border-stone-700 rounded-sm bg-stone-900 text-stone-50 hover:bg-stone-800"
        >
          Start a new adventure
        </button>
        <button
          onClick={() => navigator.clipboard?.writeText(location.href)}
          className="px-4 py-2 border border-stone-500 rounded-sm bg-white/60 hover:bg-white"
        >
          Copy link
        </button>
      </div>
    </aside>
  );
}
