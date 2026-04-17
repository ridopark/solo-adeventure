"use client";

import { useCallback, useEffect, useReducer } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { Page, StartStoryResponse, Story } from "@/lib/types";
import { useLocalStoryCache } from "./useLocalStoryCache";

type Status = "idle" | "hydrating" | "page_ready" | "choosing" | "ended" | "error";

interface State {
  status: Status;
  pages: Page[];
  stylePrefix: string;
  error?: string;
}

type Action =
  | { type: "hydrate"; story: Story }
  | { type: "seed"; start: StartStoryResponse }
  | { type: "choose_start" }
  | { type: "choose_ok"; page: Page }
  | { type: "choose_err"; error: string }
  | { type: "reset" };

const initial: State = { status: "hydrating", pages: [], stylePrefix: "" };

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case "hydrate":
      return {
        status: action.story.pages.at(-1)?.isEnding ? "ended" : "page_ready",
        pages: action.story.pages,
        stylePrefix: action.story.stylePrefix,
      };
    case "seed":
      return {
        status: action.start.page.isEnding ? "ended" : "page_ready",
        pages: [action.start.page],
        stylePrefix: action.start.stylePrefix,
      };
    case "choose_start":
      return { ...state, status: "choosing", error: undefined };
    case "choose_ok":
      return {
        status: action.page.isEnding ? "ended" : "page_ready",
        pages: [...state.pages, action.page],
        stylePrefix: state.stylePrefix,
      };
    case "choose_err":
      return { ...state, status: "error", error: action.error };
    case "reset":
      return initial;
  }
}

export function useStory(storyId: string) {
  const router = useRouter();
  const cache = useLocalStoryCache(storyId);
  const [state, dispatch] = useReducer(reducer, initial);

  useEffect(() => {
    const seedKey = `solo-adv:seed:${storyId}`;
    const seedRaw = typeof window !== "undefined" ? sessionStorage.getItem(seedKey) : null;
    if (seedRaw) {
      try {
        const seed = JSON.parse(seedRaw) as StartStoryResponse;
        dispatch({ type: "seed", start: seed });
        sessionStorage.removeItem(seedKey);
        return;
      } catch {
        /* fall through */
      }
    }
    const local = cache.read();
    if (local) {
      dispatch({ type: "hydrate", story: local });
    }
    api
      .getStory(storyId)
      .then((story) => dispatch({ type: "hydrate", story }))
      .catch((err: unknown) => {
        if (!local) {
          dispatch({
            type: "choose_err",
            error: err instanceof Error ? err.message : "failed to load",
          });
        }
      });
  }, [storyId, cache]);

  useEffect(() => {
    if (state.pages.length > 0) {
      cache.write({
        storyId,
        topic: "",
        stylePrefix: state.stylePrefix,
        pages: state.pages,
        createdAt: "",
        updatedAt: "",
      });
    }
  }, [state.pages, state.stylePrefix, storyId, cache]);

  const choose = useCallback(
    async (index: number) => {
      dispatch({ type: "choose_start" });
      try {
        const res = await api.choose(storyId, index);
        dispatch({ type: "choose_ok", page: res.page });
      } catch (err) {
        dispatch({
          type: "choose_err",
          error: err instanceof Error ? err.message : "failed",
        });
      }
    },
    [storyId],
  );

  const restart = useCallback(() => {
    cache.clear();
    router.push("/");
  }, [cache, router]);

  return {
    status: state.status,
    pages: state.pages,
    current: state.pages.at(-1) ?? null,
    error: state.error,
    choose,
    restart,
  };
}
