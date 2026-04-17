import type { Choice } from "@/lib/types";

export function ChoiceButtons({
  choices,
  disabled,
  onChoose,
}: {
  choices: Choice[];
  disabled: boolean;
  onChoose: (index: number) => void;
}) {
  return (
    <div className="space-y-3">
      {choices.map((c, i) => (
        <button
          key={i}
          onClick={() => onChoose(i)}
          disabled={disabled}
          className="w-full text-left px-5 py-3 border border-stone-400 rounded-sm bg-[#fbf6ea] hover:bg-[#f0e7cf] disabled:opacity-40 transition"
        >
          {c.label}
        </button>
      ))}
    </div>
  );
}
