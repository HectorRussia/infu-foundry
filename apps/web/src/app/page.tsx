import type { Metadata } from 'next';
import { LoginScreen } from '@/features/auth/login-screen';

export const metadata: Metadata = { title: 'เข้าสู่ระบบ' };
export default function HomePage() {
  return <LoginScreen />;
}
