export type EndingType = "victory" | "defeat" | "twist";

export interface Choice {
  label: string;
}

export interface Page {
  index: number;
  narrative: string;
  imageUrl: string | null;
  imageProvider?: string;
  audioUrl?: string;
  choices: Choice[];
  isEnding: boolean;
  endingType?: EndingType;
  createdAt: string;
}

export interface Story {
  storyId: string;
  topic: string;
  stylePrefix: string;
  pages: Page[];
  createdAt: string;
  updatedAt: string;
}

export interface StartStoryResponse {
  storyId: string;
  stylePrefix: string;
  page: Page;
}

export interface ProgressResponse {
  page: Page;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatarUrl?: string;
  createdAt: string;
}
