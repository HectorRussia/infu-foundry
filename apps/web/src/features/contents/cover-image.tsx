'use client';

import { useRef, useState } from 'react';
import { ImagePlus, Trash2, X } from 'lucide-react';
import { APIError, COVER_TYPES, MAX_COVER_BYTES, removeCover, uploadCover } from '@/lib/api';

export function CoverImage({ contentId, initialUrl }: { contentId: string; initialUrl?: string }) {
  const [url, setUrl] = useState(initialUrl ?? '');
  const [progress, setProgress] = useState<number | null>(null);
  const [message, setMessage] = useState('');
  const cancel = useRef<AbortController | null>(null);
  const input = useRef<HTMLInputElement>(null);

  async function choose(file: File) {
    if (!COVER_TYPES.includes(file.type)) {
      setMessage('รองรับเฉพาะ JPG, PNG หรือ WebP');
      return;
    }
    if (file.size > MAX_COVER_BYTES) {
      setMessage('ภาพใหญ่เกิน 10 MB');
      return;
    }
    const controller = new AbortController();
    cancel.current = controller;
    setMessage('');
    setProgress(0);
    try {
      setUrl(await uploadCover(contentId, file, setProgress, controller.signal));
      setMessage('ตั้งภาพปกแล้ว');
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') setMessage('ยกเลิกแล้ว');
      else if (error instanceof APIError && error.status === 422) {
        setMessage('ภาพนี้ใช้ไม่ได้ ตรวจชนิดและขนาดไฟล์');
      } else setMessage('อัปโหลดภาพปกไม่สำเร็จ ลองอีกครั้ง');
    } finally {
      cancel.current = null;
      setProgress(null);
      if (input.current) input.current.value = '';
    }
  }

  async function remove() {
    if (!window.confirm('ลบภาพปก?')) return;
    try {
      await removeCover(contentId);
      setUrl('');
      setMessage('ลบภาพปกแล้ว');
    } catch {
      setMessage('ลบภาพปกไม่สำเร็จ ลองอีกครั้ง');
    }
  }

  const uploading = progress !== null;

  return (
    <section aria-labelledby="cover-heading">
      <div className="mb-3 flex items-center justify-between gap-2">
        <h2 id="cover-heading" className="flex items-center gap-2 text-sm font-medium">
          ภาพปก <span className="font-normal text-muted">/ Cover</span>
        </h2>
        {url && !uploading && (
          <button
            type="button"
            onClick={remove}
            aria-label="ลบภาพปก"
            className="grid min-h-11 min-w-11 place-items-center rounded-lg text-muted hover:text-red-300"
          >
            <Trash2 size={16} />
          </button>
        )}
      </div>
      <label
        className={`group relative block aspect-video overflow-hidden rounded-2xl border border-edge bg-surface ${uploading ? 'pointer-events-none' : 'cursor-pointer hover:border-muted/60'}`}
      >
        {url ? (
          // eslint-disable-next-line @next/next/no-img-element -- presigned R2 URL, static export
          <img src={url} alt="ภาพปกของเนื้อหา" className="size-full object-cover" />
        ) : (
          <span className="grid size-full place-items-center border-dashed text-center text-sm text-muted">
            <span>
              <ImagePlus size={28} className="mx-auto mb-2" aria-hidden />
              เลือกภาพปก
              <span className="mt-1 block text-xs">JPG, PNG, WebP ไม่เกิน 10 MB</span>
            </span>
          </span>
        )}
        {url && !uploading && (
          <span className="absolute inset-x-0 bottom-0 flex items-center justify-center gap-2 bg-black/60 py-2 text-xs text-ink opacity-0 backdrop-blur transition-opacity group-hover:opacity-100 max-lg:opacity-100">
            <ImagePlus size={14} aria-hidden /> เปลี่ยนภาพปก
          </span>
        )}
        {uploading && (
          <span className="absolute inset-0 grid place-items-center bg-black/60 text-sm tabular-nums">
            {Math.round((progress ?? 0) * 100)}%
          </span>
        )}
        <input
          ref={input}
          type="file"
          accept={COVER_TYPES.join(',')}
          disabled={uploading}
          className="sr-only"
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) void choose(file);
          }}
        />
      </label>
      {uploading && (
        <button
          type="button"
          onClick={() => cancel.current?.abort()}
          className="mt-2 flex min-h-11 items-center gap-1 rounded-lg px-2 text-xs text-muted hover:text-ink"
        >
          <X size={14} /> ยกเลิก
        </button>
      )}
      {message && (
        <p role="status" className="mt-2 text-xs text-muted">
          {message}
        </p>
      )}
    </section>
  );
}
