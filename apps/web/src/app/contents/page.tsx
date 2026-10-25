import type { Metadata } from 'next';
import Link from 'next/link';
import { Plus } from 'lucide-react';

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
      <p className="mt-4 text-xs leading-6 text-muted">
        หน้าตัวอย่าง · ยังไม่มีระบบบัญชีจริงหรือบันทึกข้อมูล
      </p>
      <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
        <h2 className="font-medium">ยังไม่มีเนื้อหา</h2>
        <p className="mt-2 text-sm leading-6 text-muted">กด “เขียนเนื้อหา” เพื่อเปิดพื้นที่เขียน</p>
      </section>
    </>
  );
}
