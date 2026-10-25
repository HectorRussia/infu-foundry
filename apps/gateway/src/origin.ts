// Forwards an API request to an external Go host (e.g. Render) while the browser
// keeps calling same-origin /api/*, so production needs no CORS.
export function forwardToOrigin(
  request: Request,
  origin: string,
  secret?: string,
): Promise<Response> {
  const target = new URL(origin);
  const url = new URL(request.url);
  url.protocol = target.protocol;
  url.host = target.host;
  const forwarded = new Request(url, request);
  forwarded.headers.delete('Host');
  // Always overwrite so a client can never supply its own gateway secret.
  forwarded.headers.delete('X-Gateway-Secret');
  if (secret) forwarded.headers.set('X-Gateway-Secret', secret);
  return fetch(forwarded);
}
