export const apiBaseURL =
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  (process.env.NODE_ENV === 'development' ? 'http://localhost:8080/api/v1' : '/api/v1');

export async function checkHealth(signal: AbortSignal): Promise<void> {
  const response = await fetch(`${apiBaseURL.replace(/\/$/, '')}/health`, {
    signal,
    cache: 'no-store',
  });
  if (!response.ok) throw new Error('API unavailable');
  const data: unknown = await response.json();
  if (!data || typeof data !== 'object' || !('status' in data) || data.status !== 'ok') {
    throw new Error('Unexpected health response');
  }
}
