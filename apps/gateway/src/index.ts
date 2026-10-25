import { Container, getContainer } from '@cloudflare/containers';
import { forwardToOrigin } from './origin';
import { routeRequest } from './router';

interface Env {
  ASSETS: Fetcher;
  // Free plan: Go runs elsewhere (Render) and /api/* is forwarded to API_ORIGIN.
  API_ORIGIN?: string;
  GATEWAY_SECRET?: string;
  // Workers Paid: only bound by wrangler.containers.jsonc.
  BACKEND?: DurableObjectNamespace<Backend>;
  DATABASE_URL?: string;
}

function backendFor(env: Env) {
  return (request: Request): Promise<Response> => {
    if (env.API_ORIGIN) return forwardToOrigin(request, env.API_ORIGIN, env.GATEWAY_SECRET);
    if (env.BACKEND) return getContainer(env.BACKEND, 'infu-foundry-api').fetch(request);
    throw new Error('No API backend configured');
  };
}

export class Backend extends Container<Env> {
  defaultPort = 8080;
  sleepAfter = '10m';
  envVars = {
    APP_ENV: 'production',
    PORT: '8080',
    ...(this.env.DATABASE_URL ? { DATABASE_URL: this.env.DATABASE_URL } : {}),
  };

  override async fetch(request: Request): Promise<Response> {
    // containerFetch starts the container and waits for the port to become ready.
    // A real health response is checked before forwarding the original request.
    const health = await this.containerFetch(
      new Request('http://container/api/v1/health', {
        signal: AbortSignal.timeout(10_000),
      }),
    );
    const healthy = health.ok;
    await health.body?.cancel();
    if (!healthy) throw new Error('Backend health check failed');
    // Forward directly after readiness: transport failures must reach the
    // gateway's 503 handler, not the helper's internal-error response body.
    const url = new URL(request.url);
    url.protocol = 'http:';
    url.host = 'container';
    const forwarded = new Request(url, request);
    forwarded.headers.delete('Host');
    return this.ctx.container!.getTcpPort(this.defaultPort).fetch(forwarded);
  }
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    return routeRequest(request, backendFor(env), (incoming) => env.ASSETS.fetch(incoming));
  },
} satisfies ExportedHandler<Env>;
