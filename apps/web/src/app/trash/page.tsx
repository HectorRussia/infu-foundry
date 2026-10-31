import type { Metadata } from 'next';
import { TrashList } from '@/features/trash/trash-list';

export const metadata: Metadata = { title: 'ถังขยะ' };
export default function TrashPage() {
  return <TrashList />;
}
