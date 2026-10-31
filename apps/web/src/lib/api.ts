export const apiBaseURL =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  (process.env.NODE_ENV === 'development' ? 'http://localhost:8080/api/v1' : '/api/v1');

const baseURL = apiBaseURL.replace(/\/$/, '');

export type ContentSummary = {
  id: string;
  authorName: string;
  videoCount: number;
  // Short-lived presigned R2 link to the cover image; absent when there is none.
  coverUrl?: string;
  // CSS object-position percentages chosen in the editor (50, 50 = centre).
  coverPositionX: number;
  coverPositionY: number;
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
} & Partial<Cover> &
  Pick<Cover, 'coverPositionX' | 'coverPositionY'>;

// Cover is what the cover endpoints return. The download URL answers with
// Content-Disposition: attachment (named after the title) so phones save the image.
export type Cover = {
  coverUrl: string;
  coverDownloadUrl: string;
  coverPositionX: number;
  coverPositionY: number;
};

export type ContentInput = { title: string; body: string };

export type User = { id: string; email: string; displayName: string };

export const UNAUTHORIZED_EVENT = 'infu-foundry:unauthorized';

export class APIError extends Error {
  constructor(readonly status: number) {
    super(`API request failed with status ${status}`);
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${baseURL}${path}`, {
    ...init,
    cache: 'no-store',
    // Sends the HttpOnly session cookie (same-origin in production, :8080 in dev).
    credentials: 'include',
    headers: init.body ? { 'Content-Type': 'application/json' } : undefined,
  });
  if (response.status === 401 && !path.startsWith('/auth/')) {
    window.dispatchEvent(new Event(UNAUTHORIZED_EVENT));
  }
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

// Trashed is a deleted content: restore it, or delete it for good (its row, cover
// and videos are removed from Neon and R2). Nothing is purged automatically.
export type Trashed = {
  id: string;
  authorName: string;
  title: string;
  excerpt: string;
  videoCount: number;
  deletedAt: string;
  coverUrl?: string;
  coverPositionX: number;
  coverPositionY: number;
};

export async function listTrash(signal?: AbortSignal): Promise<Trashed[]> {
  const data = await request<{ items: Trashed[] }>('/trash', { signal });
  return data.items;
}

export function restoreContent(id: string): Promise<void> {
  return request<void>(`/trash/${encodeURIComponent(id)}/restore`, {
    method: 'POST',
    body: '{}',
  });
}

export function purgeContent(id: string): Promise<void> {
  return request<void>(`/trash/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export function emptyTrash(): Promise<void> {
  return request<void>('/trash', { method: 'DELETE' });
}

export type Media = {
  id: string;
  contentId: string;
  uploaderId: string;
  filename: string;
  contentType: string;
  sizeBytes: number;
  status: 'pending' | 'ready';
  createdAt: string;
  // Short-lived presigned R2 link (about 1 hour); reload the list to refresh it.
  url: string;
  // Same lifetime as url, but R2 sends it as an attachment so phones save the file.
  downloadUrl: string;
};

export type GalleryItem = Media & { contentTitle: string; uploaderName: string };

export async function listGallery(signal?: AbortSignal): Promise<GalleryItem[]> {
  const data = await request<{ items: GalleryItem[] }>('/media', { signal });
  return data.items;
}

export const VIDEO_TYPES = ['video/mp4', 'video/quicktime', 'video/webm'];
export const MAX_VIDEO_BYTES = 1024 ** 3;

export async function listMedia(contentId: string, signal?: AbortSignal): Promise<Media[]> {
  const data = await request<{ items: Media[] }>(
    `/contents/${encodeURIComponent(contentId)}/media`,
    { signal },
  );
  return data.items;
}

export function deleteMedia(id: string): Promise<void> {
  return request<void>(`/media/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

// putToR2 uploads a file to a presigned R2 URL with progress; XHR because fetch
// cannot report upload progress.
function putToR2(
  uploadUrl: string,
  file: File,
  onProgress: (fraction: number) => void,
  signal: AbortSignal,
): Promise<void> {
  return new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('PUT', uploadUrl);
    xhr.setRequestHeader('Content-Type', file.type);
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable) onProgress(event.loaded / event.total);
    };
    xhr.onload = () => (xhr.status < 300 ? resolve() : reject(new APIError(xhr.status)));
    xhr.onerror = () => reject(new APIError(0));
    xhr.onabort = () => reject(new DOMException('Upload cancelled', 'AbortError'));
    signal.addEventListener('abort', () => xhr.abort(), { once: true });
    xhr.send(file);
  });
}

export const COVER_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
export const MAX_COVER_BYTES = 10 * 1024 ** 2;

// uploadCover sends the image straight to R2, then asks Go to make it the cover
// (Go checks the object and deletes the previous cover). The crop starts centred.
export async function uploadCover(
  contentId: string,
  file: File,
  onProgress: (fraction: number) => void,
  signal: AbortSignal,
): Promise<Cover> {
  const path = `/contents/${encodeURIComponent(contentId)}/cover`;
  const { key, uploadUrl } = await request<{ key: string; uploadUrl: string }>(path, {
    method: 'POST',
    body: JSON.stringify({ contentType: file.type, size: file.size }),
  });
  await putToR2(uploadUrl, file, onProgress, signal);
  return request<Cover>(path, { method: 'PUT', body: JSON.stringify({ key }) });
}

// setCoverPosition saves which part of the cover stays visible when cards crop it.
export function setCoverPosition(contentId: string, x: number, y: number): Promise<Cover> {
  return request<Cover>(`/contents/${encodeURIComponent(contentId)}/cover`, {
    method: 'PATCH',
    body: JSON.stringify({ x: Math.round(x), y: Math.round(y) }),
  });
}

export function removeCover(contentId: string): Promise<void> {
  return request<void>(`/contents/${encodeURIComponent(contentId)}/cover`, { method: 'DELETE' });
}

// uploadVideo asks Go for a presigned URL, PUTs the file straight to R2 (the
// bytes never pass through the API), then confirms so Go marks it ready.
export async function uploadVideo(
  contentId: string,
  file: File,
  onProgress: (fraction: number) => void,
  signal: AbortSignal,
): Promise<Media> {
  const { media, uploadUrl } = await request<{ media: Media; uploadUrl: string }>(
    `/contents/${encodeURIComponent(contentId)}/media`,
    {
      method: 'POST',
      body: JSON.stringify({ filename: file.name, contentType: file.type, size: file.size }),
    },
  );
  await putToR2(uploadUrl, file, onProgress, signal);
  return request<Media>(`/media/${encodeURIComponent(media.id)}/complete`, {
    method: 'POST',
    body: '{}',
  });
}

export function login(email: string, password: string): Promise<User> {
  return request<User>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export function logout(): Promise<void> {
  // JSON body: the API rejects non-JSON writes as a cross-site request guard.
  return request<void>('/auth/logout', { method: 'POST', body: '{}' });
}

export function currentUser(signal?: AbortSignal): Promise<User> {
  return request<User>('/auth/me', { signal });
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
