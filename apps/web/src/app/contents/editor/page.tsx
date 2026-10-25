import type { Metadata } from 'next';
import Link from 'next/link';
import { ArrowLeft } from 'lucide-react';
import { Suspense } from 'react';
import { ContentEditor } from '@/features/contents/content-editor';

export const metadata: Metadata = { title: 'เขียนเนื้อหา' };
export default function EditorPage() {
  return (
    <>
      <Link
        href="/contents/"
        className="-ml-2 mb-3 inline-flex min-h-11 items-center gap-2 rounded-lg px-2 text-sm text-muted hover:text-ink"
      >
        <ArrowLeft size={18} />
        คลังเนื้อหา
      </Link>
      <h1 className="text-2xl font-semibold">เขียนเนื้อหา</h1>
      <Suspense fallback={<p className="mt-6 text-sm text-muted">กำลังโหลดเนื้อหา…</p>}>
        <ContentEditor />
      </Suspense>
    </>
  );
}
