export function NarrativeBlock({ text }: { text: string }) {
  return (
    <div className="text-lg leading-8 text-stone-800 whitespace-pre-wrap">
      {text}
    </div>
  );
}
