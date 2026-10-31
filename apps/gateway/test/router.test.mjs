import { test } from 'node:test';
import assert from 'node:assert/strict';
import { routeRequest } from '../src/router.ts';

test('API forwarding preserves method, query, body, headers and response cookies', async () => {
  const request = new Request('https://foundry.example/api/v1/contents?draft=1', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Cookie: 'session=test' },
    body: '{"title":"idea"}',
  });
  const response = await routeRequest(
    request,
    async (incoming) => {
      assert.equal(incoming.url, request.url);
      assert.equal(incoming.method, 'POST');
      assert.equal(incoming.headers.get('Cookie'), 'session=test');
      assert.equal(await incoming.text(), '{"title":"idea"}');
      return new Response('created', {
        status: 201,
        headers: { 'Set-Cookie': 'session=next; HttpOnly' },
      });
    },
    async () => assert.fail('API must not hit static assets'),
  );
  assert.equal(response.status, 201);
  assert.equal(response.headers.get('Set-Cookie'), 'session=next; HttpOnly');
});

test('category routes are forwarded with the rest of /api', async () => {
  const request = new Request('https://foundry.example/api/v1/categories?category=uncategorized', {
    method: 'DELETE',
  });
  const response = await routeRequest(
    request,
    async (incoming) => {
      assert.equal(incoming.url, request.url);
      assert.equal(incoming.method, 'DELETE');
      return new Response(null, { status: 204 });
    },
    async () => assert.fail('categories must not hit static assets'),
  );
  assert.equal(response.status, 204);
});

test('unknown pages retain the asset service 404, including /api-like paths', async () => {
  for (const path of ['/missing/', '/apiary/']) {
    const response = await routeRequest(
      new Request(`https://foundry.example${path}`),
      async () => assert.fail('not an API path'),
      async () => new Response('not found', { status: 404 }),
    );
    assert.equal(response.status, 404);
  }
});

test('backend failure produces a non-cached 503 and is never retried', async () => {
  let attempts = 0;
  const response = await routeRequest(
    new Request('https://foundry.example/api/v1/health'),
    async () => {
      attempts++;
      throw new Error('private details');
    },
    async () => assert.fail('no asset fallback for API'),
  );
  assert.equal(attempts, 1);
  assert.equal(response.status, 503);
  assert.equal(response.headers.get('Cache-Control'), 'no-store');
  assert.equal(response.headers.get('Retry-After'), '5');
  assert.deepEqual(await response.json(), { error: 'service_unavailable' });
});
