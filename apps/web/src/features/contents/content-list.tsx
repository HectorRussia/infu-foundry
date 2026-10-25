'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
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
    <ul className="mt-5 grid gap-3">
      {items.map((item) => (
        <li key={item.id}>
          <Link
            href={`/contents/editor/?id=${item.id}`}
            className="block rounded-2xl border border-edge bg-surface px-5 py-4 hover:border-muted/60"
          >
            <h2 className="truncate font-medium">{item.title || 'ไม่มีหัวข้อ'}</h2>
            {item.excerpt && (
              <p className="mt-1 line-clamp-2 text-sm leading-6 text-muted">{item.excerpt}</p>
            )}
            <p className="mt-3 text-xs text-muted">
              {item.authorName} · แก้ไข {dateFormat.format(new Date(item.updatedAt))}
            </p>
          </Link>
        </li>
      ))}
    </ul>
  );
}
