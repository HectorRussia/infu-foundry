import { Info } from 'lucide-react';

export function PreviewNotice({ children }: { children?: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2.5 rounded-xl border border-brand/15 bg-brand/5 p-3.5 text-sm leading-relaxed text-[#e1bba9]">
      <Info size={17} className="mt-0.5 shrink-0 text-brand" />
      <p>{children ?? 'หน้าตัวอย่าง ยังไม่มีการเข้าสู่ระบบหรือบันทึกข้อมูลจริง'}</p>
    </div>
  );
}
