type RequestHandler = (request: Request) => Promise<Response>;

export async function routeRequest(
  request: Request,
  backend: RequestHandler,
  assets: RequestHandler,
): Promise<Response> {
  const { pathname } = new URL(request.url);
  if (pathname === '/api' || pathname.startsWith('/api/')) {
    try {
      return await backend(request);
    } catch {
      console.error('Backend unavailable');
      return Response.json(
        { error: 'service_unavailable' },
        {
          status: 503,
          headers: { 'Cache-Control': 'no-store', 'Retry-After': '5' },
        },
      );
    }
  }
  return assets(request);
}
