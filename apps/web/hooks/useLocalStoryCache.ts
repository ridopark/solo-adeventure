"use client";

import { useMemo } from "react";
import type { Story } from "@/lib/types";

export function useLocalStoryCache(id: string) {
  return useMemo(() => {
    const key = `solo-adv:story:${id}`;
    return {
      read(): Story | null {
        try {
          const raw = typeof window !== "undefined" ? window.localStorage.getItem(key) : null;
          return raw ? (JSON.parse(raw) as Story) : null;
        } catch {
          return null;
        }
      },
      write(story: Story) {
        try {
          window.localStorage.setItem(key, JSON.stringify(story));
        } catch {
          /* quota or private-mode -- ignore */
        }
      },
      clear() {
        try {
          window.localStorage.removeItem(key);
        } catch {
          /* ignore */
        }
      },
    };
  }, [id]);
}
