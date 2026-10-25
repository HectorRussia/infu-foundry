export const apiBaseURL =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  (process.env.NODE_ENV === 'development' ? 'http://localhost:8080/api/v1' : '/api/v1');

const baseURL = apiBaseURL.replace(/\/$/, '');

export type ContentSummary = {
  id: string;
  authorName: string;
  title: string;
  excerpt: string;
  updatedAt: string;
};

export type Content = {
  id: string;
  authorId: string;
  authorName: string;
  title: string;
  body: string;
  createdAt: string;
  updatedAt: string;
};

export type ContentInput = { title: string; body: string };

export class APIError extends Error {
  constructor(readonly status: number) {
    super(`API request failed with status ${status}`);
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, {
    ...init,
    cache: 'no-store',
    headers: init.body ? { 'Content-Type': 'application/json' } : undefined,
  });
  if (!response.ok) throw new APIError(response.status);
  return (response.status === 204 ? undefined : await response.json()) as T;
}

export async function listContents(signal?: AbortSignal): Promise<ContentSummary[]> {
  const data = await request<{ items: ContentSummary[] }>('/contents', { signal });
  return data.items;
}

export function getContent(id: string, signal?: AbortSignal): Promise<Content> {
  return request<Content>(`/contents/${encodeURIComponent(id)}`, { signal });
}

export function saveContent(id: string | null, input: ContentInput): Promise<Content> {
  return id
    ? request<Content>(`/contents/${encodeURIComponent(id)}`, {
        method: 'PUT',
        body: JSON.stringify(input),
      })
    : request<Content>('/contents', { method: 'POST', body: JSON.stringify(input) });
}

export function deleteContent(id: string): Promise<void> {
  return request<void>(`/contents/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function checkHealth(signal: AbortSignal): Promise<void> {
  const response = await fetch(`${baseURL}/health`, {
    signal,
    cache: 'no-store',
  });
  if (!response.ok) throw new Error('API unavailable');
  const data: unknown = await response.json();
  if (!data || typeof data !== 'object' || !('status' in data) || data.status !== 'ok') {
    throw new Error('Unexpected health response');
  }
}
