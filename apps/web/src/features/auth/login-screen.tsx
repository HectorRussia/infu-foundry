import { Flame } from 'lucide-react';
import { LoginPreview } from './login-preview';

export function LoginScreen() {
  return (
    <section className="w-full max-w-sm rounded-2xl border border-edge bg-surface p-6 sm:p-8">
      <span className="mb-6 grid size-11 place-items-center rounded-xl bg-brand text-canvas">
        <Flame size={25} />
      </span>
      <h1 className="text-2xl font-semibold">ยินดีต้อนรับกลับ</h1>
      <p className="mt-2 text-sm leading-6 text-muted">เข้าสู่พื้นที่ทำงานของทีม Infu Foundry</p>
      <LoginPreview />
      <p className="mt-6 border-t border-edge pt-5 text-center text-xs leading-6 text-muted">
        สำหรับบัญชีที่ทีมสร้างให้ล่วงหน้า
      </p>
    </section>
  );
}
