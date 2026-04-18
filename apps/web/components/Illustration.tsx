"use client";

import { useState } from "react";

export function Illustration({
  src,
  alt,
  seq = 0,
}: {
  src: string | null;
  alt: string;
  seq?: number;
}) {
  const [loaded, setLoaded] = useState(false);
  if (!src) {
    return (
      <div className="aspect-square rounded-md border border-stone-300 shadow-inner bg-stone-200 flex items-center justify-center text-xs text-stone-500">
        no illustration
      </div>
    );
  }
  const variant = seq % 2 === 0 ? "ken-burns-a" : "ken-burns-b";
  return (
    <div className="aspect-square rounded-md border border-stone-300 shadow-inner bg-stone-200 overflow-hidden relative">
      {!loaded && <div className="absolute inset-0 animate-pulse bg-stone-300" />}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={src}
        alt={alt}
        onLoad={() => setLoaded(true)}
        className={`w-full h-full object-cover transition-opacity duration-500 ${loaded ? `opacity-100 ${variant}` : "opacity-0"}`}
      />
    </div>
  );
}
