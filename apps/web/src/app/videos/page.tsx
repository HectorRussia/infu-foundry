import type { Metadata } from 'next';
import { VideoGallery } from '@/features/videos/video-gallery';

export const metadata: Metadata = { title: 'รายการทั้งหมด' };
export default function VideosPage() {
  return (
    <>
      <h1 className="text-2xl font-semibold">รายการทั้งหมด</h1>
      <VideoGallery />
    </>
  );
}
