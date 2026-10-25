'use client';

import { ArrowRight } from 'lucide-react';
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { DEV_LOGIN, DEV_EMAIL, DEV_PASSWORD, loginDev } from './dev-auth';

export function LoginPreview() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const field =
    'mt-2 min-h-14 w-full rounded-xl border border-edge bg-canvas px-4 text-base text-ink placeholder:text-muted/60 focus:border-brand/60 focus:outline-none focus:ring-2 focus:ring-brand/15';
  return (
    <form
      className="mt-7 space-y-5"
      onSubmit={(event) => {
        event.preventDefault();
        if (!DEV_LOGIN) return;
        const data = new FormData(event.currentTarget);
        const message = loginDev(
          String(data.get('email') ?? ''),
          String(data.get('password') ?? ''),
        );
        setError(message);
        if (!message) router.replace('/contents/');
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
          required={DEV_LOGIN}
          inputMode="email"
          placeholder="you@example.com"
          autoComplete="off"
          aria-describedby="login-note"
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
          required={DEV_LOGIN}
          placeholder="กรอกรหัสผ่าน"
          autoComplete="off"
          aria-describedby="login-note"
          className={field}
        />
      </div>
      <button
        type="submit"
        disabled={!DEV_LOGIN}
        className="flex min-h-12 w-full items-center justify-center gap-3 rounded-xl bg-brand px-5 text-sm font-semibold text-canvas disabled:opacity-40"
      >
        เข้าสู่ระบบ
        <ArrowRight size={18} />
      </button>
      {error && (
        <p role="alert" className="text-sm text-red-300">
          {error}
        </p>
      )}
      {DEV_LOGIN ? (
        <div
          id="login-note"
          className="rounded-xl border border-edge bg-canvas p-3 text-xs leading-6 text-muted"
        >
          <p className="font-medium text-brand">บัญชีทดสอบ · Local dev เท่านั้น</p>
          <p>
            อีเมล: <code className="select-all text-ink">{DEV_EMAIL}</code>
          </p>
          <p>
            รหัสผ่าน: <code className="select-all text-ink">{DEV_PASSWORD}</code>
          </p>
          <p className="mt-1">ใช้ทดลองหน้าจอ ยังไม่ใช่ระบบบัญชีจริง</p>
        </div>
      ) : (
        <p id="login-note" className="text-center text-xs leading-6 text-muted">
          ฟอร์มตัวอย่าง ยังไม่ส่งข้อมูลหรือตรวจสอบบัญชี
          <br />
          ใช้ข้อมูลสมมติในการทดลอง
        </p>
      )}
    </form>
  );
}
