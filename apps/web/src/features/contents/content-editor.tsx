'use client';

import { useRouter, useSearchParams } from 'next/navigation';
import { useEffect, useState } from 'react';
import { Check, Copy, Save, Trash2 } from 'lucide-react';
import { APIError, deleteContent, getContent, saveContent } from '@/lib/api';

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

  return (
    <form
      className="mt-6 space-y-6"
      onSubmit={(event) => {
        event.preventDefault();
        if (dirty && !empty && !saving) void save();
      }}
    >
      <div>
        <div className="mb-3 flex items-center justify-between gap-2">
          <label htmlFor="title" className="flex items-center gap-2 text-sm font-medium">
            หัวข้อเนื้อหา
          </label>
          {id && (
            <button
              type="button"
              onClick={remove}
              className="flex min-h-11 items-center gap-2 rounded-lg px-2 text-sm text-muted hover:text-red-300"
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
          className="min-h-14 w-full rounded-xl border border-edge bg-surface px-4 py-3.5 text-base leading-relaxed text-ink placeholder:text-muted/60 focus:border-brand/60 focus:outline-none focus:ring-2 focus:ring-brand/15 sm:px-5 sm:text-lg"
        />
      </div>
      <section>
        <div className="mb-3 flex items-center justify-between gap-2">
          <label htmlFor="content" className="flex items-center gap-2 text-sm font-medium">
            เนื้อหา <span className="font-normal text-muted">/ Content</span>
          </label>
          <span className="text-xs tabular-nums text-muted">
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
            placeholder={'เริ่มเขียนตรงนี้ได้เลย…\n\nหรือวางเนื้อหาที่เตรียมไว้ แล้วคัดลอกไปใช้ต่อ'}
            className="block min-h-[42svh] w-full resize-y bg-transparent px-4 py-5 text-base leading-8 text-ink placeholder:text-muted/55 focus:outline-none sm:min-h-[420px] sm:px-5"
          />
          <div
            className="border-t border-edge/70 px-4 py-3 text-xs leading-relaxed text-muted sm:px-5"
            id="editor-storage-note"
          >
            {dirty
              ? 'มีการแก้ไขที่ยังไม่บันทึก'
              : id
                ? 'บันทึกในคลังเนื้อหาแล้ว'
                : 'กดบันทึกเพื่อเก็บลงคลังเนื้อหา'}
          </div>
        </div>
      </section>
      <div className="fixed inset-x-0 bottom-0 z-20 border-t border-edge bg-canvas/95 px-4 pt-3 pb-[calc(.75rem+env(safe-area-inset-bottom))] backdrop-blur-xl sm:px-8 lg:sticky lg:inset-x-auto lg:bottom-4 lg:rounded-2xl lg:border lg:bg-surface/95 lg:px-5">
        <div className="mx-auto max-w-6xl">
          <p
            role="status"
            aria-live="polite"
            className={`mb-2 text-center text-xs leading-relaxed lg:text-left ${copied || feedback === 'บันทึกแล้ว' ? 'text-emerald-300' : 'text-muted'}`}
          >
            {feedback || 'ทุกคนที่เข้าสู่ระบบจะเห็นเนื้อหาในคลังนี้'}
          </p>
          <div className="grid grid-cols-2 gap-3 lg:flex lg:justify-end">
            <button
              type="button"
              disabled={!content}
              onClick={copyContent}
              className="flex min-h-12 items-center justify-center gap-2 rounded-xl border border-edge bg-elevated px-4 text-sm font-medium text-ink transition-colors hover:border-muted/60 disabled:opacity-40 lg:min-w-44"
            >
              {copied ? <Check size={18} /> : <Copy size={18} />}
              {copied ? 'คัดลอกแล้ว' : 'คัดลอกเนื้อหา'}
            </button>
            <button
              type="submit"
              disabled={!dirty || empty || saving}
              aria-describedby="editor-storage-note"
              className="flex min-h-12 items-center justify-center gap-2 rounded-xl bg-brand px-4 text-sm font-semibold text-canvas disabled:opacity-40 lg:min-w-44"
            >
              <Save size={18} />
              {saving ? 'กำลังบันทึก…' : 'บันทึกเนื้อหา'}
            </button>
          </div>
        </div>
      </div>
    </form>
  );
}
