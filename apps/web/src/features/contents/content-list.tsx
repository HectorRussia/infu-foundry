'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { Film } from 'lucide-react';
import { listContents, type ContentSummary } from '@/lib/api';

const dateFormat = new Intl.DateTimeFormat('th-TH', { dateStyle: 'medium', timeStyle: 'short' });

export function ContentList() {
  const [items, setItems] = useState<ContentSummary[] | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    listContents(controller.signal)
      .then(setItems)
      .catch(() => {
        if (!controller.signal.aborted) setFailed(true);
      });
    return () => controller.abort();
  }, []);

  if (failed) {
    return (
      <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
        <h2 className="font-medium">โหลดเนื้อหาไม่ได้</h2>
        <p className="mt-2 text-sm leading-6 text-muted">
          ตรวจว่า API เปิดอยู่และเชื่อมฐานข้อมูลแล้ว จากนั้นรีเฟรชหน้านี้
        </p>
      </section>
    );
  }

  if (!items) {
    return <p className="mt-5 text-sm text-muted">กำลังโหลดเนื้อหา…</p>;
  }

  if (items.length === 0) {
    return (
      <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
        <h2 className="font-medium">ยังไม่มีเนื้อหา</h2>
        <p className="mt-2 text-sm leading-6 text-muted">กด “เขียนเนื้อหา” เพื่อเปิดพื้นที่เขียน</p>
      </section>
    );
  }

  return (
    <>
      <p className="mt-2 text-sm text-muted">{items.length.toLocaleString('th-TH')} เนื้อหา</p>
      <ul className="mt-5 grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
        {items.map((item) => {
          return (
            <li key={item.id}>
              <Link
                href={`/contents/editor/?id=${item.id}`}
                className="group flex h-full flex-col overflow-hidden rounded-2xl border border-edge bg-surface transition-colors hover:border-muted/50"
              >
                <div className="relative aspect-video overflow-hidden bg-black">
                  {item.coverUrl ? (
                    // eslint-disable-next-line @next/next/no-img-element -- presigned R2 URL, static export
                    <img
                      src={item.coverUrl}
                      alt=""
                      loading="lazy"
                      className="size-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
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
                  <h2 className="line-clamp-2 font-medium leading-snug group-hover:text-brand">
                    {item.title || 'ไม่มีหัวข้อ'}
                  </h2>
                  {item.excerpt && (
                    <p className="mt-2 line-clamp-2 text-sm leading-6 text-muted">{item.excerpt}</p>
                  )}
                  <p className="mt-auto truncate pt-3 text-xs text-muted">
                    {item.authorName} · แก้ไข {dateFormat.format(new Date(item.updatedAt))}
                  </p>
                </div>
              </Link>
            </li>
          );
        })}
      </ul>
    </>
  );
}
