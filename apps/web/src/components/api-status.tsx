'use client';

import { useEffect, useState } from 'react';
import { Activity, RefreshCw } from 'lucide-react';
import { checkHealth } from '@/lib/api';

export function APIStatus() {
  const [status, setStatus] = useState<'checking' | 'online' | 'offline'>('checking');
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let active = true;
    const timer = setTimeout(() => controller.abort(), 5000);
    checkHealth(controller.signal)
      .then(() => {
        if (active) setStatus('online');
      })
      .catch(() => {
        if (active) setStatus('offline');
      })
      .finally(() => clearTimeout(timer));
    return () => {
      active = false;
      clearTimeout(timer);
      controller.abort();
    };
  }, [attempt]);
  const labels = { checking: 'กำลังตรวจสอบ', online: 'เชื่อมต่อแล้ว', offline: 'เชื่อมต่อไม่ได้' };
  return (
    <section
      className="flex flex-wrap items-center gap-3 rounded-2xl border border-edge bg-surface p-4 sm:p-5"
      aria-label="สถานะระบบ"
    >
      <span className="hidden size-10 items-center justify-center rounded-xl bg-elevated text-muted sm:flex">
        <Activity size={20} />
      </span>
      <div className="min-w-0 flex-1">
        <strong className="text-sm font-medium">สถานะระบบ</strong>
        <p className="mt-1 text-xs text-muted">การเชื่อมต่อ API</p>
      </div>
      <span
        role="status"
        className={`inline-flex items-center gap-2 text-xs ${status === 'online' ? 'text-emerald-300' : status === 'offline' ? 'text-brand' : 'text-muted'}`}
      >
        <span className="size-1.5 rounded-full bg-current" />
        {labels[status]}
      </span>
      <button
        className="grid size-11 shrink-0 place-items-center rounded-xl border border-edge bg-elevated text-muted hover:text-ink disabled:opacity-40"
        aria-label="ตรวจสอบ API อีกครั้ง"
        disabled={status === 'checking'}
        onClick={() => {
          setStatus('checking');
          setAttempt((value) => value + 1);
        }}
      >
        <RefreshCw size={17} className={status === 'checking' ? 'motion-safe:animate-spin' : ''} />
      </button>
    </section>
  );
}
