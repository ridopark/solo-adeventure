"use client";

import { useCallback, useEffect, useReducer } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { Page, StartStoryResponse, Story } from "@/lib/types";
import { useLocalStoryCache } from "./useLocalStoryCache";

type Status = "idle" | "hydrating" | "page_ready" | "choosing" | "ended" | "error" | "needs_auth";

interface State {
  status: Status;
  pages: Page[];
  stylePrefix: string;
  cursor: number;
  error?: string;
}

type Action =
  | { type: "hydrate"; story: Story; startAt?: number }
  | { type: "seed"; start: StartStoryResponse }
  | { type: "choose_start" }
  | { type: "choose_ok"; page: Page }
  | { type: "choose_err"; error: string }
  | { type: "needs_auth" }
  | { type: "goto"; cursor: number }
  | { type: "reset" };

const initial: State = { status: "hydrating", pages: [], stylePrefix: "", cursor: 0 };

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case "hydrate": {
      const last = action.story.pages.length - 1;
      const cursor =
        typeof action.startAt === "number"
          ? Math.max(0, Math.min(action.startAt, last))
          : Math.max(0, last);
      const page = action.story.pages[cursor];
      return {
        status: page?.isEnding ? "ended" : "page_ready",
        pages: action.story.pages,
        stylePrefix: action.story.stylePrefix,
        cursor,
      };
    }
    case "seed":
      return {
        status: action.start.page.isEnding ? "ended" : "page_ready",
        pages: [action.start.page],
        stylePrefix: action.start.stylePrefix,
        cursor: 0,
      };
    case "choose_start":
      return { ...state, status: "choosing", error: undefined };
    case "choose_ok": {
      const pages = [...state.pages, action.page];
      return {
        status: action.page.isEnding ? "ended" : "page_ready",
        pages,
        stylePrefix: state.stylePrefix,
        cursor: pages.length - 1,
      };
    }
    case "choose_err":
      return { ...state, status: "error", error: action.error };
    case "needs_auth":
      return { ...state, status: "needs_auth", error: undefined };
    case "goto":
      if (action.cursor < 0 || action.cursor >= state.pages.length) return state;
      return { ...state, cursor: action.cursor, error: undefined };
    case "reset":
      return initial;
  }
}

export function useStory(storyId: string, opts: { startAt?: number } = {}) {
  const router = useRouter();
  const cache = useLocalStoryCache(storyId);
  const [state, dispatch] = useReducer(reducer, initial);
  const { startAt } = opts;

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
      dispatch({ type: "hydrate", story: local, startAt });
    }
    api
      .getStory(storyId)
      .then((story) => dispatch({ type: "hydrate", story, startAt }))
      .catch((err: unknown) => {
        if (!local) {
          dispatch({
            type: "choose_err",
            error: err instanceof Error ? err.message : "failed to load",
          });
        }
      });
  }, [storyId, cache, startAt]);

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
        const status = (err as { status?: number }).status;
        if (status === 401) {
          dispatch({ type: "needs_auth" });
          return;
        }
        dispatch({
          type: "choose_err",
          error: err instanceof Error ? err.message : "failed",
        });
      }
    },
    [storyId],
  );

  const goPrev = useCallback(() => {
    dispatch({ type: "goto", cursor: state.cursor - 1 });
  }, [state.cursor]);

  const goNext = useCallback(() => {
    dispatch({ type: "goto", cursor: state.cursor + 1 });
  }, [state.cursor]);

  const goLatest = useCallback(() => {
    dispatch({ type: "goto", cursor: state.pages.length - 1 });
  }, [state.pages.length]);

  const restart = useCallback(() => {
    cache.clear();
    router.push("/");
  }, [cache, router]);

  const atLatest = state.cursor === state.pages.length - 1;

  return {
    status: state.status,
    pages: state.pages,
    current: state.pages[state.cursor] ?? null,
    cursor: state.cursor,
    atLatest,
    canGoPrev: state.cursor > 0,
    canGoNext: state.cursor < state.pages.length - 1,
    error: state.error,
    choose,
    goPrev,
    goNext,
    goLatest,
    restart,
  };
}
