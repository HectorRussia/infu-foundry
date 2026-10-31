import assert from 'node:assert/strict';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';
import {
  ALERT_ACK_LABEL,
  CANCEL_LABEL,
  CLOSE_MS,
  PENDING_LABEL,
  canDismiss,
  closeDelayMs,
  confirmButtonLabel,
  enqueue,
  initialFocusTarget,
  requestFromAlert,
  requestFromConfirm,
  settleQueue,
} from './confirm-request.ts';

test('confirm defaults to ยกเลิก and a reversible action', () => {
  const request = requestFromConfirm({
    title: 'ลบเนื้อหา',
    message: 'ย้ายเนื้อหานี้ไปถังขยะ? กู้คืนได้จากเมนูถังขยะ',
    confirmLabel: 'ลบเนื้อหา',
  });
  assert.equal(request.kind, 'confirm');
  assert.equal(request.cancelLabel, CANCEL_LABEL);
  assert.equal(request.danger, false);
  assert.equal(request.message, 'ย้ายเนื้อหานี้ไปถังขยะ? กู้คืนได้จากเมนูถังขยะ');
  assert.equal(initialFocusTarget(request), 'confirm');
});

test('an irreversible confirm focuses ยกเลิก and keeps the caller message', () => {
  const message =
    'ลบหมวดหมู่ “เรื่อง”?\nเนื้อหาในหมวดนี้จะกลายเป็นไม่มีหมวดหมู่ แต่จะไม่ถูกลบหรือย้ายไปถังขยะ';
  const request = requestFromConfirm({
    title: 'ลบหมวดหมู่',
    message,
    confirmLabel: 'ลบหมวดหมู่',
    danger: true,
  });
  assert.equal(request.danger, true);
  assert.equal(request.cancelLabel, 'ยกเลิก');
  assert.equal(request.message, message);
  assert.equal(initialFocusTarget(request), 'cancel');
  assert.equal(confirmButtonLabel(request, 'open'), 'ลบหมวดหมู่');
  assert.equal(confirmButtonLabel(request, 'pending'), PENDING_LABEL);
  assert.equal(confirmButtonLabel(request, 'closing'), 'ลบหมวดหมู่');
});

test('a caller can replace the cancel label', () => {
  const request = requestFromConfirm({
    title: 'ลบ',
    message: 'ลบ?',
    confirmLabel: 'ลบ',
    cancelLabel: 'ไม่ลบ',
    danger: true,
  });
  assert.equal(request.cancelLabel, 'ไม่ลบ');
});

test('a notice has one acknowledge button and is not danger', () => {
  const request = requestFromAlert({ title: 'บันทึกแล้ว', message: 'เก็บลงคลังเนื้อหาแล้ว' });
  assert.equal(request.kind, 'alert');
  assert.equal(request.confirmLabel, ALERT_ACK_LABEL);
  assert.equal(request.cancelLabel, '');
  assert.equal(request.danger, false);
  assert.equal(initialFocusTarget(request), 'confirm');
  assert.equal(
    requestFromAlert({ title: 'เสร็จ', message: 'เรียบร้อย', confirmLabel: 'ปิด' }).confirmLabel,
    'ปิด',
  );
});

test('esc and the backdrop are ignored while the action is in flight or closing', () => {
  assert.equal(canDismiss('open'), true);
  assert.equal(canDismiss('pending'), false);
  assert.equal(canDismiss('closing'), false);
});

test('motion is skipped when the user prefers reduced motion', () => {
  assert.equal(closeDelayMs(false), CLOSE_MS);
  assert.equal(closeDelayMs(true), 0);
});

test('a second request waits until the open one settles', () => {
  const first = { id: 'a' };
  const second = { id: 'b' };
  let state: { active: { id: string } | null; waiting: { id: string }[] } = {
    active: null,
    waiting: [],
  };
  state = enqueue(state, first);
  state = enqueue(state, second);
  assert.equal(state.active, first);
  assert.deepEqual(state.waiting, [second]);
  state = settleQueue(state);
  assert.equal(state.active, second);
  assert.deepEqual(state.waiting, []);
  state = settleQueue(state);
  assert.equal(state.active, null);
  assert.deepEqual(state.waiting, []);
});

test('web src does not call window.alert, window.confirm, or window.prompt', () => {
  const src = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
  const pattern = /window\.(alert|confirm|prompt)\s*\(/;
  const hits = walk(src).flatMap((file) =>
    pattern.test(readFileSync(file, 'utf8')) ? [path.relative(src, file)] : [],
  );
  assert.deepEqual(hits, []);
});

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const full = path.join(dir, name);
    if (statSync(full).isDirectory()) return walk(full);
    return /\.(ts|tsx)$/.test(name) && !name.endsWith('.test.ts') ? [full] : [];
  });
}
