'use client';

import { ArrowRight } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { APIError, login } from '@/lib/api';
import { setSignedIn, useSession } from './session';

function errorMessage(error: unknown) {
  if (error instanceof APIError && error.status === 401) return 'อีเมลหรือรหัสผ่านไม่ถูกต้อง';
  if (error instanceof APIError && error.status === 429) {
    return 'ลองผิดหลายครั้งเกินไป กรุณารอ 15 นาทีแล้วลองใหม่';
  }
  return 'เชื่อมต่อระบบไม่ได้ กรุณาลองอีกครั้ง';
}

export function LoginForm() {
  const router = useRouter();
  const session = useSession();
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const field =
    'mt-2 min-h-14 w-full rounded-xl border border-edge bg-canvas px-4 text-base text-ink placeholder:text-muted/60 focus:border-brand/60 focus:outline-none focus:ring-2 focus:ring-brand/15';

  useEffect(() => {
    if (session.status === 'signed-in') router.replace('/contents/');
  }, [router, session.status]);

  return (
    <form
      className="mt-7 space-y-5"
      onSubmit={async (event) => {
        event.preventDefault();
        const data = new FormData(event.currentTarget);
        setSubmitting(true);
        setError(null);
        try {
          setSignedIn(
            await login(String(data.get('email') ?? ''), String(data.get('password') ?? '')),
          );
        } catch (failure) {
          setError(errorMessage(failure));
          setSubmitting(false);
        }
      }}
    >
      <div>
        <label htmlFor="email" className="text-sm font-medium">
          อีเมล
        </label>
        <input
          id="email"
          name="email"
          type="email"
          required
          inputMode="email"
          placeholder="you@example.com"
          autoComplete="username"
          className={field}
        />
      </div>
      <div>
        <label htmlFor="password" className="text-sm font-medium">
          รหัสผ่าน
        </label>
        <input
          id="password"
          name="password"
          type="password"
          required
          placeholder="กรอกรหัสผ่าน"
          autoComplete="current-password"
          className={field}
        />
      </div>
      <button
        type="submit"
        disabled={submitting}
        className="flex min-h-12 w-full items-center justify-center gap-3 rounded-xl bg-brand px-5 text-sm font-semibold text-canvas disabled:opacity-40"
      >
        {submitting ? 'กำลังเข้าสู่ระบบ…' : 'เข้าสู่ระบบ'}
        <ArrowRight size={18} />
      </button>
      {submitting && (
        <p className="text-center text-xs leading-6 text-muted">
          ถ้าไม่ได้ใช้งานสักพัก ระบบอาจใช้เวลาตื่นประมาณครึ่งนาที
        </p>
      )}
      {error && (
        <p role="alert" className="text-sm text-red-300">
          {error}
        </p>
      )}
    </form>
  );
}
