'use client';

import { useId, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react';
import { Check, Crosshair, ImagePlus, Move, Trash2, X } from 'lucide-react';
import {
  APIError,
  COVER_TYPES,
  MAX_COVER_BYTES,
  removeCover,
  setCoverPosition,
  uploadCover,
  type Cover,
} from '@/lib/api';
import { DownloadLink } from '@/features/videos/download-link';

type Position = { x: number; y: number };

const clamp = (value: number) => Math.min(100, Math.max(0, value));

export function CoverImage({ contentId, initial }: { contentId: string; initial?: Cover }) {
  const [cover, setCover] = useState<Cover | undefined>(initial);
  // While adjusting, `draft` is the unsaved crop; otherwise the saved one is shown.
  const [draft, setDraft] = useState<Position | null>(null);
  const [savingPosition, setSavingPosition] = useState(false);
  const [progress, setProgress] = useState<number | null>(null);
  const [message, setMessage] = useState('');
  const cancel = useRef<AbortController | null>(null);
  const input = useRef<HTMLInputElement>(null);
  const image = useRef<HTMLImageElement>(null);
  const drag = useRef<{ pointerX: number; pointerY: number; start: Position } | null>(null);
  const inputId = useId();

  const adjusting = draft !== null;
  const uploading = progress !== null;
  const position = draft ?? { x: cover?.coverPositionX ?? 50, y: cover?.coverPositionY ?? 50 };

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
      setCover(await uploadCover(contentId, file, setProgress, controller.signal));
      setMessage('ตั้งภาพปกแล้ว กด “จัดตำแหน่ง” เพื่อเลือกส่วนที่จะโชว์');
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
      setCover(undefined);
      setMessage('ลบภาพปกแล้ว');
    } catch {
      setMessage('ลบภาพปกไม่สำเร็จ ลองอีกครั้ง');
    }
  }

  async function savePosition() {
    if (!draft) return;
    setSavingPosition(true);
    try {
      setCover(await setCoverPosition(contentId, draft.x, draft.y));
      setDraft(null);
      setMessage('บันทึกตำแหน่งภาพปกแล้ว');
    } catch {
      setMessage('บันทึกตำแหน่งไม่สำเร็จ ลองอีกครั้ง');
    } finally {
      setSavingPosition(false);
    }
  }

  // How far the cropped image overflows the frame, in pixels. Dragging by the full
  // overflow moves object-position from 0% to 100%; 0 means that axis cannot move.
  function overflow() {
    const el = image.current;
    if (!el?.naturalWidth || !el.naturalHeight) return { x: 0, y: 0 };
    const scale = Math.max(el.clientWidth / el.naturalWidth, el.clientHeight / el.naturalHeight);
    return {
      x: el.naturalWidth * scale - el.clientWidth,
      y: el.naturalHeight * scale - el.clientHeight,
    };
  }

  function startDrag(event: PointerEvent<HTMLDivElement>) {
    if (!adjusting) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    drag.current = { pointerX: event.clientX, pointerY: event.clientY, start: position };
  }

  function moveDrag(event: PointerEvent<HTMLDivElement>) {
    if (!drag.current) return;
    const { x: overX, y: overY } = overflow();
    const { pointerX, pointerY, start } = drag.current;
    // Dragging the picture right reveals more of its left side, so the % goes down.
    setDraft({
      x: overX > 0 ? clamp(start.x - ((event.clientX - pointerX) / overX) * 100) : start.x,
      y: overY > 0 ? clamp(start.y - ((event.clientY - pointerY) / overY) * 100) : start.y,
    });
  }

  function nudge(event: KeyboardEvent<HTMLDivElement>) {
    if (!adjusting) return;
    const step = event.shiftKey ? 10 : 2;
    const moves: Record<string, Position> = {
      ArrowLeft: { x: -step, y: 0 },
      ArrowRight: { x: step, y: 0 },
      ArrowUp: { x: 0, y: -step },
      ArrowDown: { x: 0, y: step },
    };
    const move = moves[event.key];
    if (move) {
      event.preventDefault();
      setDraft({ x: clamp(position.x + move.x), y: clamp(position.y + move.y) });
    } else if (event.key === 'Enter') void savePosition();
    else if (event.key === 'Escape') setDraft(null);
  }

  function startAdjusting() {
    const { x, y } = overflow();
    setMessage(
      x < 1 && y < 1
        ? 'ภาพนี้สัดส่วนพอดีกรอบแล้ว ไม่มีส่วนที่ถูกครอปให้เลื่อน'
        : 'ลากภาพเพื่อเลือกส่วนที่จะโชว์บนการ์ด แล้วกดบันทึก',
    );
    setDraft(position);
  }

  const picture = cover && (
    // eslint-disable-next-line @next/next/no-img-element -- presigned R2 URL, static export
    <img
      ref={image}
      src={cover.coverUrl}
      alt="ภาพปกของเนื้อหา"
      draggable={false}
      style={{ objectPosition: `${position.x}% ${position.y}%` }}
      className="size-full select-none object-cover"
    />
  );

  return (
    <section aria-labelledby="cover-heading">
      <div className="mb-3 flex items-center justify-between gap-2">
        <h2 id="cover-heading" className="flex items-center gap-2 text-sm font-medium">
          ภาพปก <span className="font-normal text-muted">/ Cover</span>
        </h2>
        {cover && !uploading && !adjusting && (
          <div className="flex items-center">
            <button
              type="button"
              onClick={startAdjusting}
              className="flex min-h-11 items-center gap-1.5 rounded-lg px-3 text-xs text-muted transition-colors hover:bg-elevated hover:text-ink"
            >
              <Move size={16} aria-hidden /> จัดตำแหน่ง
            </button>
            <DownloadLink href={cover.coverDownloadUrl} filename="ภาพปก" />
            <button
              type="button"
              onClick={remove}
              aria-label="ลบภาพปก"
              className="grid min-h-11 min-w-11 place-items-center rounded-lg text-muted hover:text-red-300"
            >
              <Trash2 size={16} />
            </button>
          </div>
        )}
      </div>

      {adjusting ? (
        <div
          role="application"
          aria-label="ลากหรือกดลูกศรเพื่อจัดตำแหน่งภาพปก Enter บันทึก Esc ยกเลิก"
          tabIndex={0}
          onPointerDown={startDrag}
          onPointerMove={moveDrag}
          onPointerUp={() => (drag.current = null)}
          onPointerCancel={() => (drag.current = null)}
          onKeyDown={nudge}
          // touch-action: none keeps a finger drag from scrolling the page on phones.
          className="relative aspect-video cursor-grab touch-none overflow-hidden rounded-2xl border border-brand bg-surface outline-none active:cursor-grabbing focus-visible:ring-2 focus-visible:ring-brand"
        >
          {picture}
          <span className="pointer-events-none absolute inset-0 grid place-items-center">
            <span className="flex items-center gap-1.5 rounded-full bg-black/60 px-3 py-1.5 text-xs text-ink backdrop-blur">
              <Move size={14} aria-hidden /> ลากเพื่อเลื่อนภาพ
            </span>
          </span>
        </div>
      ) : (
        <label
          htmlFor={inputId}
          className={`group relative block aspect-video overflow-hidden rounded-2xl border border-edge bg-surface ${uploading ? 'pointer-events-none' : 'cursor-pointer hover:border-muted/60'}`}
        >
          {picture ?? (
            <span className="grid size-full place-items-center border-dashed text-center text-sm text-muted">
              <span>
                <ImagePlus size={28} className="mx-auto mb-2" aria-hidden />
                เลือกภาพปก
                <span className="mt-1 block text-xs">JPG, PNG, WebP ไม่เกิน 10 MB</span>
              </span>
            </span>
          )}
          {cover && !uploading && (
            <span className="absolute inset-x-0 bottom-0 flex items-center justify-center gap-2 bg-black/60 py-2 text-xs text-ink opacity-0 backdrop-blur transition-opacity group-hover:opacity-100 max-lg:opacity-100">
              <ImagePlus size={14} aria-hidden /> เปลี่ยนภาพปก
            </span>
          )}
          {uploading && (
            <span className="absolute inset-0 grid place-items-center bg-black/60 text-sm tabular-nums">
              {Math.round((progress ?? 0) * 100)}%
            </span>
          )}
        </label>
      )}
      <input
        ref={input}
        id={inputId}
        type="file"
        accept={COVER_TYPES.join(',')}
        disabled={uploading || adjusting}
        className="sr-only"
        onChange={(event) => {
          const file = event.target.files?.[0];
          if (file) void choose(file);
        }}
      />

      {adjusting && (
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={savePosition}
            disabled={savingPosition}
            className="flex min-h-11 items-center gap-1.5 rounded-lg bg-brand px-4 text-xs font-medium text-canvas disabled:opacity-60"
          >
            <Check size={16} aria-hidden /> {savingPosition ? 'กำลังบันทึก…' : 'บันทึกตำแหน่ง'}
          </button>
          <button
            type="button"
            onClick={() => setDraft({ x: 50, y: 50 })}
            className="flex min-h-11 items-center gap-1.5 rounded-lg px-3 text-xs text-muted hover:bg-elevated hover:text-ink"
          >
            <Crosshair size={16} aria-hidden /> กึ่งกลาง
          </button>
          <button
            type="button"
            onClick={() => {
              setDraft(null);
              setMessage('');
            }}
            className="flex min-h-11 items-center gap-1.5 rounded-lg px-3 text-xs text-muted hover:bg-elevated hover:text-ink"
          >
            <X size={16} aria-hidden /> ยกเลิก
          </button>
        </div>
      )}
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
