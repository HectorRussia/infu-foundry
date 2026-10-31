import type { Metadata, Viewport } from 'next';
import { AppShell } from '@/components/app-shell';
import { ConfirmProvider } from '@/components/confirm-dialog';
import '@fontsource-variable/noto-sans-thai';
import './globals.css';

export const metadata: Metadata = {
  title: { default: 'Infu Foundry — พื้นที่สำหรับไอเดียของทีม', template: '%s | Infu Foundry' },
  description:
    'A creative workspace for organizing ideas and content — from first draft to published work.',
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  viewportFit: 'cover',
  themeColor: '#101114',
  colorScheme: 'dark',
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="th">
      <body>
        <ConfirmProvider>
          <AppShell>{children}</AppShell>
        </ConfirmProvider>
      </body>
    </html>
  );
}
