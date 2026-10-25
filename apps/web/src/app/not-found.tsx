import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="py-20 text-center">
      <p className="text-xs tracking-[.2em] text-brand">404 / NOT FOUND</p>
      <h1 className="mt-4 text-3xl font-semibold">ยังไม่มีหน้านี้</h1>
      <p className="mt-4 text-sm text-muted">กลับไปเริ่มต้นที่พื้นที่ทำงานของคุณ</p>
      <Link
        href="/"
        className="mt-7 inline-flex min-h-12 items-center justify-center rounded-xl bg-brand px-6 text-sm font-semibold text-canvas"
      >
        กลับหน้าแรก
      </Link>
    </div>
  );
}
