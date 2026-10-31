'use client';

import { useEffect, useState } from 'react';
import { Film, RotateCcw, Trash2 } from 'lucide-react';
import { useConfirm } from '@/components/confirm-dialog';
import { emptyTrash, listTrash, purgeContent, restoreContent, type Trashed } from '@/lib/api';

const dateFormat = new Intl.DateTimeFormat('th-TH', { dateStyle: 'medium', timeStyle: 'short' });

export function TrashList() {
  const confirm = useConfirm();
  const [items, setItems] = useState<Trashed[] | null>(null);
  const [failed, setFailed] = useState(false);
  // id of the item being restored/deleted, or 'all' while emptying the trash.
  const [busy, setBusy] = useState<string | null>(null);
  const [message, setMessage] = useState('');

  useEffect(() => {
    const controller = new AbortController();
    listTrash(controller.signal)
      .then(setItems)
      .catch(() => {
        if (!controller.signal.aborted) setFailed(true);
      });
    return () => controller.abort();
  }, []);

  const name = (item: Trashed) => item.title || 'ไม่มีหัวข้อ';

  async function act(item: Trashed, action: 'restore' | 'purge') {
    const run = async () => {
      setBusy(item.id);
      setMessage('');
      try {
        await (action === 'restore' ? restoreContent(item.id) : purgeContent(item.id));
        setItems((current) => current?.filter(({ id }) => id !== item.id) ?? null);
        setMessage(
          action === 'restore'
            ? `กู้คืน "${name(item)}" กลับไปที่คลังเนื้อหาแล้ว`
            : `ลบ "${name(item)}" ถาวรแล้ว`,
        );
      } catch {
        setMessage(
          action === 'restore' ? 'กู้คืนไม่สำเร็จ ลองอีกครั้ง' : 'ลบถาวรไม่สำเร็จ ลองอีกครั้ง',
        );
      } finally {
        setBusy(null);
      }
    };
    if (action === 'restore') {
      await run();
      return;
    }
    await confirm({
      title: 'ลบถาวร',
      message: `ลบ "${name(item)}" ถาวร?\nภาพปกและวิดีโอทั้งหมดจะถูกลบด้วย กู้คืนไม่ได้`,
      confirmLabel: 'ลบถาวร',
      danger: true,
      onConfirm: run,
    });
  }

  async function empty() {
    if (!items?.length) return;
    const current = items;
    await confirm({
      title: 'ล้างถังขยะ',
      message: `ล้างถังขยะ ${current.length} รายการ?\nภาพปกและวิดีโอทั้งหมดจะถูกลบด้วย กู้คืนไม่ได้`,
      confirmLabel: 'ล้างถังขยะ',
      danger: true,
      onConfirm: async () => {
        setBusy('all');
        setMessage('');
        try {
          await emptyTrash();
          setItems([]);
          setMessage('ล้างถังขยะแล้ว');
        } catch {
          // Some items may already be gone; reload to show what is left.
          setItems(await listTrash().catch(() => current));
          setMessage('ล้างถังขยะไม่ครบ ลองอีกครั้ง');
        } finally {
          setBusy(null);
        }
      },
    });
  }

  return (
    <>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold">ถังขยะ</h1>
        {Boolean(items?.length) && (
          <button
            type="button"
            onClick={empty}
            disabled={busy !== null}
            className="flex min-h-12 items-center justify-center gap-2 rounded-xl px-4 text-sm text-red-300 transition-colors hover:bg-red-400/10 disabled:opacity-50"
          >
            <Trash2 size={18} />
            {busy === 'all' ? 'กำลังล้าง…' : 'ล้างถังขยะ'}
          </button>
        )}
      </div>
      <p className="mt-2 text-sm text-muted">
        เนื้อหาที่ลบจะอยู่ที่นี่จนกว่าจะกู้คืนหรือลบถาวร
        การลบถาวรจะลบภาพปกและวิดีโอของเนื้อหานั้นด้วย
      </p>
      {message && (
        <p
          role="status"
          className="mt-4 rounded-xl border border-edge bg-surface px-4 py-3 text-sm"
        >
          {message}
        </p>
      )}

      {failed ? (
        <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
          <h2 className="font-medium">โหลดถังขยะไม่ได้</h2>
          <p className="mt-2 text-sm leading-6 text-muted">
            ตรวจว่า API เปิดอยู่ แล้วรีเฟรชหน้านี้
          </p>
        </section>
      ) : !items ? (
        <p className="mt-5 text-sm text-muted">กำลังโหลดถังขยะ…</p>
      ) : items.length === 0 ? (
        <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
          <h2 className="font-medium">ถังขยะว่าง</h2>
        </section>
      ) : (
        <ul className="mt-5 grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {items.map((item) => (
            <li
              key={item.id}
              className="flex flex-col overflow-hidden rounded-2xl border border-edge bg-surface"
            >
              <div className="relative aspect-video overflow-hidden bg-black opacity-70 grayscale">
                {item.coverUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element -- presigned R2 URL, static export
                  <img
                    src={item.coverUrl}
                    alt=""
                    loading="lazy"
                    style={{ objectPosition: `${item.coverPositionX}% ${item.coverPositionY}%` }}
                    className="size-full object-cover"
                  />
                ) : (
                  <div className="size-full bg-gradient-to-br from-elevated to-canvas p-5">
                    <p className="line-clamp-5 text-xs leading-6 text-muted/70">
                      {item.excerpt || item.title || 'ไม่มีข้อความ'}
                    </p>
                  </div>
                )}
                {item.videoCount > 0 && (
                  <span className="absolute right-3 bottom-3 flex items-center gap-1 rounded-full bg-black/70 px-2.5 py-1 text-[11px] text-ink backdrop-blur">
                    <Film size={12} aria-hidden /> {item.videoCount} วิดีโอ
                  </span>
                )}
              </div>
              <div className="flex flex-1 flex-col p-4">
                <h2 className="line-clamp-2 font-medium leading-snug">{name(item)}</h2>
                <p className="mt-auto truncate pt-3 text-xs text-muted">
                  {item.authorName} · ลบเมื่อ {dateFormat.format(new Date(item.deletedAt))}
                </p>
                <div className="mt-3 grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => act(item, 'restore')}
                    disabled={busy !== null}
                    className="flex min-h-11 items-center justify-center gap-1.5 rounded-xl bg-elevated text-sm transition-colors hover:text-brand disabled:opacity-50"
                  >
                    <RotateCcw size={16} aria-hidden /> กู้คืน
                  </button>
                  <button
                    type="button"
                    onClick={() => act(item, 'purge')}
                    disabled={busy !== null}
                    className="flex min-h-11 items-center justify-center gap-1.5 rounded-xl text-sm text-red-300 transition-colors hover:bg-red-400/10 disabled:opacity-50"
                  >
                    <Trash2 size={16} aria-hidden /> {busy === item.id ? 'กำลังทำ…' : 'ลบถาวร'}
                  </button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
