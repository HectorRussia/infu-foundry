'use client';

import { useState, type FormEvent } from 'react';
import { APIError, createCategory, deleteCategory, renameCategory, type Category } from '@/lib/api';

const chip = 'inline-flex min-h-11 max-w-full items-center gap-2 rounded-full border px-4 text-sm';
const field =
  'min-h-12 w-full rounded-xl border border-edge bg-surface px-4 text-base text-ink placeholder:text-muted/60 focus:border-brand/60 focus:outline-none focus:ring-2 focus:ring-brand/15';
const action =
  'inline-flex min-h-12 w-full shrink-0 items-center justify-center rounded-xl border border-edge bg-elevated px-4 text-sm font-medium disabled:opacity-40 sm:w-auto';

export function CategoryBar({
  categories,
  filter,
  onFilter,
  onReload,
}: {
  categories: Category[];
  filter: string;
  onFilter: (category: string) => void;
  onReload: () => void;
}) {
  const selected = categories.find((category) => category.id === filter);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [draft, setDraft] = useState<{ id: string; name: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');
  const rename = draft && draft.id === selected?.id ? draft.name : (selected?.name ?? '');

  async function create(event: FormEvent) {
    event.preventDefault();
    const next = name.trim();
    if (!next || busy) return;
    setBusy(true);
    setMessage('');
    try {
      const category = await createCategory(next);
      setName('');
      setCreating(false);
      setMessage('');
      onFilter(category.id);
      onReload();
    } catch (error) {
      setMessage(
        error instanceof APIError && error.status === 409
          ? 'มีหมวดชื่อนี้อยู่แล้ว'
          : 'สร้างหมวดหมู่ไม่สำเร็จ',
      );
    } finally {
      setBusy(false);
    }
  }

  async function saveName(event: FormEvent) {
    event.preventDefault();
    if (!selected || busy) return;
    const next = rename.trim();
    if (!next || next === selected.name) return;
    setBusy(true);
    setMessage('');
    try {
      await renameCategory(selected.id, next);
      setDraft(null);
      setMessage('เปลี่ยนชื่อแล้ว');
      onReload();
    } catch (error) {
      setMessage(
        error instanceof APIError && error.status === 409
          ? 'มีหมวดชื่อนี้อยู่แล้ว'
          : 'เปลี่ยนชื่อไม่สำเร็จ',
      );
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!selected || busy) return;
    const ok = window.confirm(
      `ลบหมวดหมู่ “${selected.name}”?\nเนื้อหาในหมวดนี้จะกลายเป็นไม่มีหมวดหมู่ แต่จะไม่ถูกลบหรือย้ายไปถังขยะ`,
    );
    if (!ok) return;
    setBusy(true);
    setMessage('');
    try {
      await deleteCategory(selected.id);
      onFilter('');
      onReload();
    } catch {
      setMessage('ลบหมวดหมู่ไม่สำเร็จ');
    } finally {
      setBusy(false);
    }
  }

  const filters: { value: string; label: string }[] = [
    { value: '', label: 'ทั้งหมด' },
    { value: 'uncategorized', label: 'ไม่มีหมวดหมู่' },
  ];

  return (
    <section aria-label="หมวดหมู่" className="mt-5">
      <div role="group" aria-label="กรองตามหมวดหมู่" className="flex flex-wrap gap-2">
        {filters.map(({ value, label }) => (
          <button
            key={label}
            type="button"
            aria-pressed={filter === value}
            onClick={() => onFilter(value)}
            className={`${chip} ${filter === value ? 'border-brand/50 bg-brand/15 text-brand' : 'border-edge bg-surface text-muted hover:text-ink'}`}
          >
            {label}
          </button>
        ))}
        {categories.map((category) => (
          <button
            key={category.id}
            type="button"
            aria-pressed={filter === category.id}
            onClick={() => onFilter(category.id)}
            className={`${chip} ${filter === category.id ? 'border-brand/50 bg-brand/15 text-brand' : 'border-edge bg-surface text-muted hover:text-ink'}`}
          >
            <span className="truncate">{category.name}</span>
            <span className="tabular-nums text-xs opacity-70">
              {category.contentCount.toLocaleString('th-TH')}
            </span>
          </button>
        ))}
        {!creating && (
          <button
            type="button"
            onClick={() => {
              setCreating(true);
              setMessage('');
            }}
            className={`${chip} border-dashed border-edge bg-transparent text-muted hover:text-ink`}
          >
            + หมวดหมู่
          </button>
        )}
      </div>

      {creating && (
        <form onSubmit={create} className="mt-3 flex flex-col gap-2 sm:flex-row">
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            maxLength={80}
            autoComplete="off"
            placeholder="ชื่อหมวดหมู่ใหม่…"
            aria-label="ชื่อหมวดหมู่ใหม่"
            className={field}
          />
          <button type="submit" disabled={busy || !name.trim()} className={action}>
            {busy ? 'กำลังสร้าง…' : 'สร้างหมวดหมู่'}
          </button>
          <button
            type="button"
            onClick={() => {
              setCreating(false);
              setName('');
            }}
            className={action}
          >
            ยกเลิก
          </button>
        </form>
      )}

      {selected && (
        <form onSubmit={saveName} className="mt-3 flex flex-col gap-2 sm:flex-row">
          <input
            value={rename}
            onChange={(event) => {
              if (selected) setDraft({ id: selected.id, name: event.target.value });
            }}
            maxLength={80}
            autoComplete="off"
            aria-label="ชื่อหมวดหมู่"
            className={field}
          />
          <button
            type="submit"
            disabled={busy || !rename.trim() || rename.trim() === selected.name}
            className={action}
          >
            เปลี่ยนชื่อ
          </button>
          <button
            type="button"
            onClick={() => void remove()}
            disabled={busy}
            className={`${action} text-red-300 hover:bg-red-400/10`}
          >
            ลบหมวดหมู่
          </button>
        </form>
      )}

      {filter && filter !== 'uncategorized' && !selected && (
        <p className="mt-3 text-sm text-muted">ไม่พบหมวดหมู่นี้</p>
      )}
      {message && (
        <p role="status" className="mt-3 text-sm text-muted">
          {message}
        </p>
      )}
    </section>
  );
}
