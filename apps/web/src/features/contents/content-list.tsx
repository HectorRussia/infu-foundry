'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { Film } from 'lucide-react';
import { listContents, listGallery, type ContentSummary } from '@/lib/api';

const dateFormat = new Intl.DateTimeFormat('th-TH', { dateStyle: 'medium', timeStyle: 'short' });

// Latest video URL per content, used as the card cover. The gallery is optional:
// without R2 (or on failure) cards fall back to a text cover.
async function videoCovers(signal: AbortSignal) {
  const covers = new Map<string, string>();
  try {
    for (const item of await listGallery(signal)) {
      if (!covers.has(item.contentId)) covers.set(item.contentId, item.url);
    }
  } catch {
    // Text covers only.
  }
  return covers;
}

export function ContentList() {
  const [items, setItems] = useState<ContentSummary[] | null>(null);
  const [covers, setCovers] = useState(new Map<string, string>());
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    listContents(controller.signal)
      .then((list) => {
        setItems(list);
        if (list.some((item) => item.videoCount > 0)) {
          void videoCovers(controller.signal).then((found) => {
            if (!controller.signal.aborted) setCovers(found);
          });
        }
      })
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
          const cover = covers.get(item.id);
          return (
            <li key={item.id}>
              <Link
                href={`/contents/editor/?id=${item.id}`}
                className="group flex h-full flex-col overflow-hidden rounded-2xl border border-edge bg-surface transition-colors hover:border-muted/50"
              >
                <div className="relative aspect-video overflow-hidden bg-black">
                  {cover ? (
                    // First frame as a still cover; #t nudges browsers to paint it.
                    <video
                      src={`${cover}#t=0.1`}
                      muted
                      playsInline
                      preload="metadata"
                      tabIndex={-1}
                      aria-hidden
                      className="pointer-events-none size-full object-contain"
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
