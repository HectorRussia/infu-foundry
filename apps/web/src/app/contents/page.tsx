import type { Metadata } from 'next';
import Link from 'next/link';
import { Suspense } from 'react';
import { Plus } from 'lucide-react';
import { ContentList } from '@/features/contents/content-list';

export const metadata: Metadata = { title: 'คลังเนื้อหา' };
export default function ContentsPage() {
  return (
    <>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold">คลังเนื้อหา</h1>
        <Link
          href="/contents/editor/"
          className="flex min-h-12 items-center justify-center gap-2 rounded-xl bg-brand px-4 text-sm font-semibold text-canvas hover:bg-[#ffa383]"
        >
          <Plus size={18} />
          เขียนเนื้อหา
        </Link>
      </div>
      <Suspense fallback={<p className="mt-5 text-sm text-muted">กำลังโหลดเนื้อหา…</p>}>
        <ContentList />
      </Suspense>
    </>
  );
}
