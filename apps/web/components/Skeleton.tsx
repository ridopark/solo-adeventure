export function Skeleton({ variant }: { variant: "full" | "next" }) {
  if (variant === "next") {
    return (
      <div className="space-y-3">
        <div className="h-4 bg-stone-300 animate-pulse rounded" />
        <div className="h-4 bg-stone-300 animate-pulse rounded w-11/12" />
        <div className="h-4 bg-stone-300 animate-pulse rounded w-8/12" />
      </div>
    );
  }
  return (
    <div className="space-y-6">
      <div className="aspect-square bg-stone-300 animate-pulse rounded-md" />
      <div className="space-y-3">
        <div className="h-4 bg-stone-300 animate-pulse rounded" />
        <div className="h-4 bg-stone-300 animate-pulse rounded w-11/12" />
        <div className="h-4 bg-stone-300 animate-pulse rounded w-10/12" />
        <div className="h-4 bg-stone-300 animate-pulse rounded w-9/12" />
      </div>
    </div>
  );
}
