'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Flame } from 'lucide-react';

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname().replace(/\/$/, '') || '/';
  const login = pathname === '/' || pathname === '/login';
  const editor = pathname === '/contents/editor';

  if (login) {
    return <main className="grid min-h-dvh place-items-center px-4 py-6 sm:py-10">{children}</main>;
  }

  return (
    <div className="min-h-dvh">
      <header className="border-b border-edge pt-[env(safe-area-inset-top)]">
        <div className="mx-auto flex min-h-16 max-w-3xl items-center justify-between gap-4 px-4 sm:px-6">
          <Link
            href="/contents/"
            className="flex min-h-11 items-center gap-2 text-sm font-semibold"
          >
            <Flame size={22} className="text-brand" />
            Infu Foundry
          </Link>
          <Link
            href="/"
            className="inline-flex min-h-11 items-center text-xs text-muted hover:text-ink"
          >
            หน้าเข้าสู่ระบบ
          </Link>
        </div>
      </header>
      <main
        className={`mx-auto max-w-3xl px-4 pt-6 sm:px-6 sm:pt-8 ${editor ? 'pb-[calc(9rem+env(safe-area-inset-bottom))]' : 'pb-10'}`}
      >
        {children}
      </main>
    </div>
  );
}
