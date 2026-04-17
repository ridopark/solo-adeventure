import { BACKEND_URL } from "./env";
import type { ProgressResponse, StartStoryResponse, Story } from "./types";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BACKEND_URL}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`HTTP ${res.status}: ${body || res.statusText}`);
  }
  return (await res.json()) as T;
}

export const api = {
  startStory: (topic: string) =>
    request<StartStoryResponse>("/stories", {
      method: "POST",
      body: JSON.stringify({ topic }),
    }),
  choose: (storyId: string, choiceIndex: number) =>
    request<ProgressResponse>(`/stories/${storyId}/choose`, {
      method: "POST",
      body: JSON.stringify({ choiceIndex }),
    }),
  getStory: (storyId: string) => request<Story>(`/stories/${storyId}`),
  visit: (path: string) =>
    request<{ status: string }>("/visit", {
      method: "POST",
      body: JSON.stringify({
        path,
        referrer: typeof document !== "undefined" ? document.referrer : "",
        userAgent: typeof navigator !== "undefined" ? navigator.userAgent : "",
      }),
    }).catch(() => ({ status: "ignored" })),
};
