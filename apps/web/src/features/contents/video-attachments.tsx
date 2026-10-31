'use client';

import { useEffect, useRef, useState } from 'react';
import { Film, Trash2, Upload, X } from 'lucide-react';
import { useConfirm } from '@/components/confirm-dialog';
import { DownloadLink } from '@/features/videos/download-link';
import {
  APIError,
  deleteMedia,
  listMedia,
  MAX_VIDEO_BYTES,
  uploadVideo,
  VIDEO_TYPES,
  type Media,
} from '@/lib/api';

function formatSize(bytes: number) {
  return bytes >= 1024 ** 3
    ? `${(bytes / 1024 ** 3).toFixed(1)} GB`
    : `${Math.max(0.1, bytes / 1024 ** 2).toFixed(1)} MB`;
}

export function VideoAttachments({ contentId }: { contentId: string }) {
  const confirm = useConfirm();
  const [items, setItems] = useState<Media[] | null>(null);
  const [message, setMessage] = useState('');
  const [upload, setUpload] = useState<{ name: string; progress: number } | null>(null);
  const cancel = useRef<AbortController | null>(null);
  const input = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const controller = new AbortController();
    listMedia(contentId, controller.signal)
      .then(setItems)
      .catch(() => {
        if (!controller.signal.aborted) setMessage('โหลดวิดีโอไม่ได้ ลองรีเฟรชหน้านี้');
      });
    return () => controller.abort();
  }, [contentId]);

  async function attach(file: File) {
    if (!VIDEO_TYPES.includes(file.type)) {
      setMessage('รองรับเฉพาะไฟล์ MP4, MOV หรือ WebM');
      return;
    }
    if (file.size > MAX_VIDEO_BYTES) {
      setMessage('ไฟล์ใหญ่เกิน 1 GB');
      return;
    }
    const controller = new AbortController();
    cancel.current = controller;
    setMessage('');
    setUpload({ name: file.name, progress: 0 });
    try {
      const media = await uploadVideo(
        contentId,
        file,
        (progress) => setUpload({ name: file.name, progress }),
        controller.signal,
      );
      setItems((current) => [...(current ?? []), media]);
      setMessage('แนบวิดีโอแล้ว');
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') setMessage('ยกเลิกแล้ว');
      else if (error instanceof APIError && error.status === 422) {
        setMessage('ไฟล์นี้ใช้ไม่ได้ ตรวจชนิดและขนาดไฟล์');
      } else setMessage('อัปโหลดไม่สำเร็จ ลองอีกครั้ง');
    } finally {
      cancel.current = null;
      setUpload(null);
      if (input.current) input.current.value = '';
    }
  }

  async function remove(media: Media) {
    await confirm({
      title: 'ลบวิดีโอ',
      message: `ลบวิดีโอ "${media.filename}" ถาวร? กู้คืนไม่ได้`,
      confirmLabel: 'ลบวิดีโอ',
      danger: true,
      onConfirm: async () => {
        try {
          await deleteMedia(media.id);
          setItems((current) => current?.filter((item) => item.id !== media.id) ?? null);
          setMessage('ลบวิดีโอแล้ว');
        } catch {
          setMessage('ลบวิดีโอไม่สำเร็จ ลองอีกครั้ง');
        }
      },
    });
  }

  return (
    <section aria-labelledby="videos-heading">
      <div className="mb-3 flex items-center justify-between gap-2">
        <h2 id="videos-heading" className="flex items-center gap-2 text-sm font-medium">
          วิดีโอ <span className="font-normal text-muted">/ Videos</span>
        </h2>
        <label
          className={`flex min-h-11 items-center gap-2 rounded-lg px-3 text-sm ${upload ? 'pointer-events-none opacity-40' : 'cursor-pointer text-brand hover:bg-brand/10'}`}
        >
          <Upload size={16} />
          แนบวิดีโอ
          <input
            ref={input}
            type="file"
            accept={VIDEO_TYPES.join(',')}
            disabled={Boolean(upload)}
            className="sr-only"
            onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) void attach(file);
            }}
          />
        </label>
      </div>

      {upload && (
        <div className="mb-3 rounded-2xl border border-edge bg-surface p-4">
          <div className="flex items-center justify-between gap-3 text-sm">
            <span className="truncate">{upload.name}</span>
            <button
              type="button"
              onClick={() => cancel.current?.abort()}
              className="flex min-h-11 shrink-0 items-center gap-1 rounded-lg px-2 text-muted hover:text-ink"
            >
              <X size={16} />
              ยกเลิก
            </button>
          </div>
          <div
            role="progressbar"
            aria-label="ความคืบหน้าการอัปโหลด"
            aria-valuenow={Math.round(upload.progress * 100)}
            aria-valuemin={0}
            aria-valuemax={100}
            className="mt-2 h-2 overflow-hidden rounded-full bg-elevated"
          >
            <div
              className="h-full bg-brand transition-[width]"
              style={{ width: `${upload.progress * 100}%` }}
            />
          </div>
          <p className="mt-2 text-xs tabular-nums text-muted">
            {Math.round(upload.progress * 100)}% · อย่าปิดหน้านี้ระหว่างอัปโหลด
          </p>
        </div>
      )}

      {items === null && !message && <p className="text-sm text-muted">กำลังโหลดวิดีโอ…</p>}
      {items?.length === 0 && !upload && (
        <p className="rounded-2xl border border-dashed border-edge px-4 py-6 text-center text-sm text-muted">
          ยังไม่มีวิดีโอ · รองรับ MP4, MOV, WebM ไม่เกิน 1 GB
        </p>
      )}
      {items && items.length > 0 && (
        <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
          {items.map((item) => (
            <li key={item.id} className="overflow-hidden rounded-2xl border border-edge bg-surface">
              <video
                src={item.url}
                controls
                playsInline
                preload="metadata"
                className="aspect-video w-full bg-black"
              />
              <div className="flex items-center justify-between gap-2 px-4 py-2 text-xs text-muted">
                <span className="flex min-w-0 items-center gap-2">
                  <Film size={14} className="shrink-0" />
                  <span className="truncate">{item.filename}</span>
                  <span className="shrink-0 tabular-nums">{formatSize(item.sizeBytes)}</span>
                </span>
                <span className="flex shrink-0 items-center">
                  <DownloadLink href={item.downloadUrl} filename={item.filename} />
                  <button
                    type="button"
                    onClick={() => remove(item)}
                    aria-label={`ลบวิดีโอ ${item.filename}`}
                    className="grid min-h-11 min-w-11 place-items-center rounded-lg hover:text-red-300"
                  >
                    <Trash2 size={16} />
                  </button>
                </span>
              </div>
            </li>
          ))}
        </ul>
      )}
      {message && (
        <p role="status" className="mt-2 text-xs text-muted">
          {message}
        </p>
      )}
    </section>
  );
}
