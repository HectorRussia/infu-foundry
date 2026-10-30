'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { ArrowUpRight, Film } from 'lucide-react';
import { listGallery, type GalleryItem } from '@/lib/api';

const dateFormat = new Intl.DateTimeFormat('th-TH', { dateStyle: 'medium' });

export function VideoGallery() {
  const [items, setItems] = useState<GalleryItem[] | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    listGallery(controller.signal)
      .then(setItems)
      .catch(() => {
        if (!controller.signal.aborted) setFailed(true);
      });
    return () => controller.abort();
  }, []);

  if (failed) {
    return (
      <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
        <h2 className="font-medium">โหลดวิดีโอไม่ได้</h2>
        <p className="mt-2 text-sm leading-6 text-muted">ลองรีเฟรชหน้านี้อีกครั้ง</p>
      </section>
    );
  }

  if (!items) return <p className="mt-5 text-sm text-muted">กำลังโหลดวิดีโอ…</p>;

  if (items.length === 0) {
    return (
      <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
        <Film size={28} className="mx-auto text-muted" aria-hidden />
        <h2 className="mt-4 font-medium">ยังไม่มีวิดีโอ</h2>
        <p className="mt-2 text-sm leading-6 text-muted">
          เปิดเนื้อหาใน{' '}
          <Link href="/contents/" className="text-brand hover:underline">
            คลังเนื้อหา
          </Link>{' '}
          แล้วกด “แนบวิดีโอ”
        </p>
      </section>
    );
  }

  return (
    <>
      <p className="mt-2 text-sm text-muted">{items.length.toLocaleString('th-TH')} วิดีโอ</p>
      <ul className="mt-5 grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
        {items.map((item) => (
          <li
            key={item.id}
            className="group overflow-hidden rounded-2xl border border-edge bg-surface transition-colors hover:border-muted/50"
          >
            <video
              src={item.url}
              controls
              playsInline
              preload="metadata"
              className="aspect-video w-full bg-black object-contain"
            />
            <div className="p-4">
              <Link
                href={`/contents/editor/?id=${item.contentId}`}
                className="flex items-start justify-between gap-3 font-medium leading-snug hover:text-brand"
              >
                <span className="line-clamp-2">{item.contentTitle || 'ไม่มีหัวข้อ'}</span>
                <ArrowUpRight
                  size={18}
                  aria-hidden
                  className="mt-0.5 shrink-0 text-muted transition-colors group-hover:text-brand"
                />
              </Link>
              <p className="mt-2 truncate text-xs text-muted">
                {item.uploaderName} · {dateFormat.format(new Date(item.createdAt))} ·{' '}
                {item.filename}
              </p>
            </div>
          </li>
        ))}
      </ul>
    </>
  );
}
