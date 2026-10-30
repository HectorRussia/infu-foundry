'use client';

import { useRouter, useSearchParams } from 'next/navigation';
import { useEffect, useState } from 'react';
import { Check, Clock, Copy, PenLine, Save, Trash2 } from 'lucide-react';
import { APIError, deleteContent, getContent, saveContent } from '@/lib/api';
import { VideoAttachments } from './video-attachments';

const dateFormat = new Intl.DateTimeFormat('th-TH', { dateStyle: 'medium', timeStyle: 'short' });

export function ContentEditor() {
  const id = useSearchParams().get('id');
  // Remount per content so switching to "new" or another item never keeps stale text.
  return <Editor key={id ?? 'new'} id={id} />;
}

function Editor({ id }: { id: string | null }) {
  const router = useRouter();
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [saved, setSaved] = useState({ title: '', content: '' });
  const [loading, setLoading] = useState(Boolean(id));
  const [missing, setMissing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [copied, setCopied] = useState(false);
  const [meta, setMeta] = useState<{ author: string; updatedAt: string } | null>(null);
  const dirty = title !== saved.title || content !== saved.content;
  const empty = !title.trim() && !content.trim();

  useEffect(() => {
    if (!id) return;
    const controller = new AbortController();
    getContent(id, controller.signal)
      .then((item) => {
        setTitle(item.title);
        setContent(item.body);
        setSaved({ title: item.title, content: item.body });
        setMeta({ author: item.authorName, updatedAt: item.updatedAt });
        setLoading(false);
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        setLoading(false);
        if (error instanceof APIError && error.status === 404) setMissing(true);
        else setFeedback('โหลดเนื้อหาไม่ได้ ตรวจว่า API เปิดอยู่แล้วรีเฟรชหน้านี้');
      });
    return () => controller.abort();
  }, [id]);

  async function copyContent() {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setFeedback('คัดลอกเนื้อหาแล้ว พร้อมนำไปใช้ต่อ');
    } catch {
      setCopied(false);
      setFeedback('คัดลอกไม่ได้ กรุณาเลือกข้อความแล้วคัดลอกด้วยตัวเอง');
    }
  }

  async function save() {
    setSaving(true);
    try {
      const item = await saveContent(id, { title, body: content });
      setTitle(item.title);
      setSaved({ title: item.title, content: item.body });
      setMeta({ author: item.authorName, updatedAt: item.updatedAt });
      setFeedback('บันทึกแล้ว');
      if (!id) router.replace(`/contents/editor/?id=${item.id}`);
    } catch {
      setFeedback('บันทึกไม่สำเร็จ ข้อความยังอยู่ในหน้านี้ ลองกดบันทึกอีกครั้ง');
    } finally {
      setSaving(false);
    }
  }

  async function remove() {
    if (!id || !window.confirm('ลบเนื้อหานี้ออกจากคลัง?')) return;
    try {
      await deleteContent(id);
      router.replace('/contents/');
    } catch {
      setFeedback('ลบไม่สำเร็จ ลองอีกครั้ง');
    }
  }

  if (loading) return <p className="mt-6 text-sm text-muted">กำลังโหลดเนื้อหา…</p>;
  if (missing) {
    return (
      <section className="mt-6 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
        <h2 className="font-medium">ไม่พบเนื้อหานี้</h2>
        <p className="mt-2 text-sm leading-6 text-muted">
          อาจถูกลบไปแล้ว กลับไปเลือกจากคลังเนื้อหา
        </p>
      </section>
    );
  }

  const status = dirty
    ? 'มีการแก้ไขที่ยังไม่บันทึก'
    : id
      ? 'บันทึกในคลังเนื้อหาแล้ว'
      : 'กดบันทึกเพื่อเก็บลงคลังเนื้อหา';
  const message = feedback || 'ทุกคนที่เข้าสู่ระบบจะเห็นเนื้อหาในคลังนี้';
  const positive = copied || feedback === 'บันทึกแล้ว';
  const copyButton = (
    <button
      type="button"
      disabled={!content}
      onClick={copyContent}
      className="flex min-h-12 items-center justify-center gap-2 rounded-xl border border-edge bg-elevated px-4 text-sm font-medium text-ink transition-colors hover:border-muted/60 disabled:opacity-40"
    >
      {copied ? <Check size={18} /> : <Copy size={18} />}
      {copied ? 'คัดลอกแล้ว' : 'คัดลอกเนื้อหา'}
    </button>
  );
  const saveButton = (
    <button
      type="submit"
      disabled={!dirty || empty || saving}
      className="flex min-h-12 items-center justify-center gap-2 rounded-xl bg-brand px-4 text-sm font-semibold text-canvas transition-colors hover:bg-[#ffa383] disabled:opacity-40"
    >
      <Save size={18} />
      {saving ? 'กำลังบันทึก…' : 'บันทึกเนื้อหา'}
    </button>
  );

  // Mobile: one column with a fixed bottom action bar. Desktop: writing area on the
  // left and a sticky panel for status, actions and details on the right.
  return (
    <form
      className="mt-6 lg:grid lg:grid-cols-[minmax(0,1fr)_18rem] lg:items-start lg:gap-8 xl:grid-cols-[minmax(0,1fr)_20rem]"
      onSubmit={(event) => {
        event.preventDefault();
        if (dirty && !empty && !saving) void save();
      }}
    >
      <div className="space-y-6">
        <div>
          <div className="mb-3 flex items-center justify-between gap-2">
            <label htmlFor="title" className="flex items-center gap-2 text-sm font-medium">
              หัวข้อเนื้อหา
            </label>
            {id && (
              <button
                type="button"
                onClick={remove}
                className="flex min-h-11 items-center gap-2 rounded-lg px-2 text-sm text-muted hover:text-red-300 lg:hidden"
              >
                <Trash2 size={16} />
                ลบเนื้อหา
              </button>
            )}
          </div>
          <input
            id="title"
            value={title}
            onChange={(event) => {
              setTitle(event.target.value);
              setFeedback('');
            }}
            maxLength={300}
            autoComplete="off"
            placeholder="ตั้งชื่อให้ไอเดียนี้…"
            className="min-h-14 w-full rounded-xl border border-edge bg-surface px-4 py-3.5 text-base leading-relaxed text-ink placeholder:text-muted/60 focus:border-brand/60 focus:outline-none focus:ring-2 focus:ring-brand/15 sm:px-5 sm:text-lg lg:min-h-16 lg:px-7 lg:text-xl lg:font-medium"
          />
        </div>
        <section>
          <div className="mb-3 flex items-center justify-between gap-2">
            <label htmlFor="content" className="flex items-center gap-2 text-sm font-medium">
              เนื้อหา <span className="font-normal text-muted">/ Content</span>
            </label>
            <span className="text-xs tabular-nums text-muted lg:hidden">
              {content.length.toLocaleString('th-TH')} ตัวอักษร
            </span>
          </div>
          <div className="overflow-hidden rounded-2xl border border-edge bg-surface focus-within:border-brand/60 focus-within:ring-2 focus-within:ring-brand/15">
            <textarea
              id="content"
              value={content}
              onChange={(event) => {
                setContent(event.target.value);
                setCopied(false);
                setFeedback('');
              }}
              aria-describedby="editor-storage-note"
              placeholder={
                'เริ่มเขียนตรงนี้ได้เลย…\n\nหรือวางเนื้อหาที่เตรียมไว้ แล้วคัดลอกไปใช้ต่อ'
              }
              className="block min-h-[42svh] w-full resize-y bg-transparent px-4 py-5 text-base leading-8 text-ink placeholder:text-muted/55 focus:outline-none sm:min-h-[420px] sm:px-5 lg:min-h-[58vh] lg:px-7 lg:py-6"
            />
            <div
              className="border-t border-edge/70 px-4 py-3 text-xs leading-relaxed text-muted sm:px-5 lg:hidden"
              id="editor-storage-note"
            >
              {status}
            </div>
          </div>
        </section>
      </div>

      {/* Videos sit below the writing area on mobile and below the action card on desktop. */}
      <aside aria-label="การบันทึก ข้อมูลเนื้อหา และวิดีโอ" className="mt-6 space-y-6 lg:mt-0">
        <div className="hidden rounded-2xl border border-edge bg-surface p-5 lg:block">
          <p className="flex items-center gap-2 text-sm font-medium">
            <span
              aria-hidden
              className={`size-2 shrink-0 rounded-full ${dirty ? 'bg-amber-300' : id ? 'bg-emerald-400' : 'bg-muted'}`}
            />
            {status}
          </p>
          <p
            role="status"
            aria-live="polite"
            className={`mt-2 min-h-5 text-xs leading-relaxed ${positive ? 'text-emerald-300' : 'text-muted'}`}
          >
            {message}
          </p>
          <div className="mt-5 grid gap-3">
            {saveButton}
            {copyButton}
          </div>
          <dl className="mt-6 space-y-3 border-t border-edge pt-5 text-xs text-muted">
            <div className="flex items-center justify-between gap-3">
              <dt>ตัวอักษร</dt>
              <dd className="tabular-nums text-ink">{content.length.toLocaleString('th-TH')}</dd>
            </div>
            {meta && (
              <>
                <div className="flex items-center justify-between gap-3">
                  <dt className="flex items-center gap-1.5">
                    <PenLine size={13} aria-hidden /> ผู้เขียน
                  </dt>
                  <dd className="truncate text-ink">{meta.author}</dd>
                </div>
                <div className="flex items-center justify-between gap-3">
                  <dt className="flex items-center gap-1.5">
                    <Clock size={13} aria-hidden /> แก้ไขล่าสุด
                  </dt>
                  <dd className="text-right text-ink">
                    {dateFormat.format(new Date(meta.updatedAt))}
                  </dd>
                </div>
              </>
            )}
          </dl>
          {id && (
            <button
              type="button"
              onClick={remove}
              className="mt-5 flex min-h-11 w-full items-center justify-center gap-2 rounded-xl text-sm text-muted transition-colors hover:bg-red-400/10 hover:text-red-300"
            >
              <Trash2 size={16} />
              ลบเนื้อหา
            </button>
          )}
        </div>
        {id ? (
          <VideoAttachments contentId={id} />
        ) : (
          <p className="rounded-2xl border border-dashed border-edge px-4 py-4 text-center text-xs text-muted">
            บันทึกเนื้อหาก่อน แล้วจึงแนบวิดีโอได้
          </p>
        )}
      </aside>

      <div className="fixed inset-x-0 bottom-0 z-20 border-t border-edge bg-canvas/95 px-4 pt-3 pb-[calc(.75rem+env(safe-area-inset-bottom))] backdrop-blur-xl sm:px-8 lg:hidden">
        <p
          className={`mb-2 text-center text-xs leading-relaxed ${positive ? 'text-emerald-300' : 'text-muted'}`}
        >
          {message}
        </p>
        <div className="grid grid-cols-2 gap-3">
          {copyButton}
          {saveButton}
        </div>
      </div>
    </form>
  );
}
