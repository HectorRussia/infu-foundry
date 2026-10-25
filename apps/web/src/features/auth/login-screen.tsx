import { Flame } from 'lucide-react';
import { LoginPreview } from './login-preview';

export function LoginScreen() {
  return (
    <section className="w-full max-w-md">
      <div className="mb-8 text-center">
        <span className="mx-auto mb-5 grid size-14 place-items-center rounded-2xl bg-brand/10 text-brand">
          <Flame size={30} />
        </span>
        <p className="mb-2 text-[10px] tracking-[.2em] text-brand">WELCOME TO THE FOUNDRY</p>
        <h1 className="text-3xl font-semibold">กลับมาเขียนด้วยกัน</h1>
        <p className="mt-3 text-sm text-muted">พื้นที่ทำงานของทีม Infu Foundry</p>
      </div>
      <div className="rounded-2xl border border-edge bg-surface p-7">
        <div className="flex items-center justify-between gap-3">
          <h2 className="font-medium">เข้าสู่ระบบ</h2>
          <span className="rounded-full border border-edge px-2.5 py-1 text-[10px] text-muted">
            หน้าตัวอย่าง
          </span>
        </div>
        <LoginPreview />
      </div>
      <p className="mt-5 text-center text-xs leading-6 text-muted">
        สำหรับบัญชีที่ทีมสร้างให้ล่วงหน้า
      </p>
    </section>
  );
}
