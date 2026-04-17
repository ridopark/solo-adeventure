import { BACKEND_URL } from "./env";
import type { ProgressResponse, StartStoryResponse, Story, User } from "./types";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BACKEND_URL}${path}`, {
    credentials: "include",
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    const err = new Error(`HTTP ${res.status}: ${body || res.statusText}`);
    (err as { status?: number }).status = res.status;
    throw err;
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
  claim: (storyId: string) =>
    request<{ status: string }>(`/stories/${storyId}/claim`, { method: "POST" }),
  myStories: () => request<{ stories: Story[] }>(`/stories`),
  me: () => request<User>(`/auth/me`),
  logout: () => request<{ status: string }>(`/auth/logout`, { method: "POST" }),
  loginURL: (returnTo?: string) => {
    const u = new URL(`${BACKEND_URL}/auth/google/start`);
    if (returnTo) u.searchParams.set("return_to", returnTo);
    return u.toString();
  },
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
