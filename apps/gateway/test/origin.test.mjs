import { test } from 'node:test';
import assert from 'node:assert/strict';
import { forwardToOrigin } from '../src/origin.ts';

test('forwards to the external origin with path, query, body and the gateway secret', async (t) => {
  let seen;
  t.mock.method(globalThis, 'fetch', async (request) => {
    seen = request;
    return new Response('ok');
  });
  const request = new Request('https://foundry.example/api/v1/contents?draft=1', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Gateway-Secret': 'forged' },
    body: '{"title":"idea"}',
  });
  await forwardToOrigin(request, 'https://infu-api.onrender.com', 'real-secret');
  assert.equal(seen.url, 'https://infu-api.onrender.com/api/v1/contents?draft=1');
  assert.equal(seen.method, 'POST');
  assert.equal(seen.headers.get('X-Gateway-Secret'), 'real-secret');
  assert.equal(await seen.text(), '{"title":"idea"}');
});

test('a client-supplied gateway secret is stripped when none is configured', async (t) => {
  let seen;
  t.mock.method(globalThis, 'fetch', async (request) => {
    seen = request;
    return new Response('ok');
  });
  await forwardToOrigin(
    new Request('https://foundry.example/api/v1/health', {
      headers: { 'X-Gateway-Secret': 'forged' },
    }),
    'https://infu-api.onrender.com',
  );
  assert.equal(seen.headers.get('X-Gateway-Secret'), null);
});
