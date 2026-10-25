'use client';

import { ArrowRight } from 'lucide-react';

export function LoginPreview() {
  const field =
    'mt-2 min-h-14 w-full rounded-xl border border-edge bg-canvas px-4 text-base text-ink placeholder:text-muted/60 focus:border-brand/60 focus:outline-none focus:ring-2 focus:ring-brand/15';
  return (
    <form className="mt-7 space-y-5" onSubmit={(event) => event.preventDefault()}>
      <div>
        <label htmlFor="email" className="text-sm font-medium">
          อีเมล
        </label>
        <input
          id="email"
          name="email"
          type="email"
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
          placeholder="กรอกรหัสผ่าน"
          autoComplete="off"
          aria-describedby="login-note"
          className={field}
        />
      </div>
      <button
        type="submit"
        disabled
        className="flex min-h-12 w-full items-center justify-center gap-3 rounded-xl bg-brand px-5 text-sm font-semibold text-canvas disabled:opacity-40"
      >
        เข้าสู่ระบบ
        <ArrowRight size={18} />
      </button>
      <p id="login-note" className="text-center text-xs leading-6 text-muted">
        ฟอร์มตัวอย่าง ยังไม่ส่งข้อมูลหรือตรวจสอบบัญชี
        <br />
        ใช้ข้อมูลสมมติในการทดลอง
      </p>
    </form>
  );
}
