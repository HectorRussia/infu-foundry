'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { BookOpen, Flame, LogIn, LogOut, Menu, PenLine } from 'lucide-react';
import { DEV_LOGIN, hasDevSession, logoutDev, useDevSession } from '@/features/auth/dev-auth';

const navigation = [
  { href: '/contents/', label: 'คลังเนื้อหา', icon: BookOpen },
  { href: '/contents/editor/', label: 'เขียนเนื้อหา', icon: PenLine },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const session = useDevSession();
  const pathname = usePathname().replace(/\/$/, '') || '/';
  const workspace = pathname === '/contents' || pathname === '/contents/editor';
  const editor = pathname === '/contents/editor';
  useEffect(() => {
    if (DEV_LOGIN && workspace && !session && !hasDevSession()) router.replace('/login/');
  }, [router, workspace, session]);
  const exitWorkspace = DEV_LOGIN ? (
    <button
      type="button"
      onClick={() => {
        logoutDev();
        router.replace('/login/');
      }}
      className="flex min-h-12 w-full items-center gap-3 rounded-xl bg-elevated px-4 text-sm"
    >
      <LogOut size={18} />
      ออกจากระบบ
    </button>
  ) : (
    <Link href="/" className="flex min-h-12 items-center gap-3 rounded-xl bg-elevated px-4 text-sm">
      <LogIn size={18} />
      หน้าเข้าสู่ระบบ
    </Link>
  );
  const active = (href: string) => pathname === href.replace(/\/$/, '');
  const links = navigation.map(({ href, label, icon: Icon }) => (
    <Link
      key={href}
      href={href}
      aria-current={active(href) ? 'page' : undefined}
      className={`flex min-h-12 items-center gap-3 rounded-xl px-4 text-sm ${active(href) ? 'bg-brand/10 text-brand' : 'text-muted hover:bg-elevated hover:text-ink'}`}
    >
      <Icon size={19} />
      {label}
    </Link>
  ));

  if (!workspace) {
    return <main className="grid min-h-dvh place-items-center px-4 py-4 sm:py-10">{children}</main>;
  }

  if (DEV_LOGIN && !session) {
    return (
      <main className="grid min-h-dvh place-items-center text-sm text-muted">
        กำลังเปิดหน้าเข้าสู่ระบบ…
      </main>
    );
  }

  return (
    <div className="min-h-dvh">
      <aside className="fixed inset-y-0 left-0 hidden w-60 flex-col overflow-y-auto border-r border-edge bg-[#131418] px-3 py-7 lg:flex">
        <Link href="/contents/" className="flex min-h-12 items-center gap-3 text-lg font-semibold">
          <span className="grid size-10 place-items-center rounded-xl bg-brand text-canvas">
            <Flame size={24} />
          </span>
          <span>
            infu <span className="font-normal text-muted">foundry</span>
          </span>
        </Link>
        <div className="mt-10 flex items-center gap-3 rounded-xl border border-edge bg-surface p-4">
          <span className="grid size-9 shrink-0 place-items-center rounded-lg bg-elevated text-xs text-brand">
            IF
          </span>
          <div>
            <p className="text-sm font-semibold">พื้นที่ทำงานของทีม</p>
            <p className="mt-1 text-xs text-muted">Infu Foundry</p>
          </div>
        </div>
        <p className="mb-4 mt-8 px-4 text-[10px] tracking-[.2em] text-muted">WORKSPACE</p>
        <nav aria-label="เมนูหลัก" className="space-y-2">
          {links}
        </nav>
        <div className="mt-auto pt-8">
          <div className="mb-6 rounded-xl border border-edge p-4 text-xs leading-6 text-muted">
            <p className="mb-2 text-brand">พื้นที่สำหรับไอเดียถัดไป</p>
            คิดไว้ เขียนไว้
            <br />
            แล้วกลับมาต่อยอดด้วยกัน
          </div>
          {exitWorkspace}
          <p className="mt-5 flex justify-between border-t border-edge pt-5 text-[10px] text-muted">
            <span>INFU FOUNDRY</span>
            <span>V0.1</span>
          </p>
        </div>
      </aside>
      <div className="lg:pl-60">
        <header className="border-b border-edge pt-[env(safe-area-inset-top)]">
          <div className="flex min-h-16 items-center justify-between gap-4 px-4 sm:px-8 lg:min-h-20 lg:px-10">
            <span className="text-sm text-muted">
              <span className="hidden sm:inline">
                Workspace <span className="mx-3 text-edge">/</span>
              </span>
              <span className="text-ink">{editor ? 'เขียนเนื้อหา' : 'คลังเนื้อหา'}</span>
            </span>
            <span className="rounded-full border border-edge bg-surface px-2.5 py-1 text-[10px] text-muted">
              {DEV_LOGIN ? 'LOCAL DEV' : 'PREVIEW'}
            </span>
          </div>
          <details key={pathname} className="border-t border-edge px-4 lg:hidden">
            <summary className="flex min-h-12 cursor-pointer list-none items-center gap-2 text-sm text-muted [&::-webkit-details-marker]:hidden">
              <Menu size={20} />
              เมนูพื้นที่ทำงาน
            </summary>
            <nav aria-label="เมนูมือถือ" className="space-y-2 pb-3">
              {links}
              {exitWorkspace}
            </nav>
          </details>
        </header>
        <main
          className={`mx-auto max-w-4xl px-4 pt-6 sm:px-8 lg:px-10 lg:pt-10 ${editor ? 'pb-[calc(9rem+env(safe-area-inset-bottom))]' : 'pb-10'}`}
        >
          {children}
        </main>
      </div>
    </div>
  );
}
