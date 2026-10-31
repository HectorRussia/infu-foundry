import assert from 'node:assert/strict';
import { test } from 'node:test';
import { afterIdForDrop, afterIdForStep, reorder, withPinned } from './library-order.ts';

const row = (id: string, pinned = false) => ({ id, pinned });

test('drop and step stay inside the pin group', () => {
  const items = [row('p1', true), row('p2', true), row('a'), row('b'), row('c')];

  assert.equal(afterIdForStep(items, 'p2', 1), undefined);
  assert.equal(afterIdForStep(items, 'a', -1), undefined);
  assert.equal(afterIdForStep(items, 'b', -1), null);
  assert.equal(afterIdForStep(items, 'b', 1), 'c');
  assert.equal(afterIdForDrop(items, 'c', 'a', false), null);
  assert.equal(afterIdForDrop(items, 'a', 'c', true), 'c');
  assert.equal(afterIdForDrop(items, 'b', 'a', true), undefined);
  assert.equal(afterIdForDrop(items, 'a', 'p2', true), undefined);
});

test('reorder and pin match the visible library', () => {
  const items = [row('p', true), row('a'), row('b'), row('c')];
  assert.deepEqual(
    reorder(items, 'c', null).map((item) => item.id),
    ['p', 'c', 'a', 'b'],
  );
  assert.deepEqual(
    reorder(items, 'a', 'c').map((item) => item.id),
    ['p', 'b', 'c', 'a'],
  );
  assert.deepEqual(
    withPinned(items, 'b', true).map((item) => item.id),
    ['b', 'p', 'a', 'c'],
  );
  assert.equal(withPinned(items, 'b', true)[0].pinned, true);
  assert.deepEqual(
    withPinned(items, 'p', false).map((item) => item.id),
    ['p', 'a', 'b', 'c'],
  );
  assert.equal(withPinned(items, 'p', false)[0].pinned, false);
});

test('a filtered list keeps the order of the rows it still shows', () => {
  const visible = [row('a'), row('c')];
  const afterId = afterIdForStep(visible, 'c', -1);
  assert.equal(afterId, null);
  assert.deepEqual(
    reorder(visible, 'c', afterId).map((item) => item.id),
    ['c', 'a'],
  );
});
