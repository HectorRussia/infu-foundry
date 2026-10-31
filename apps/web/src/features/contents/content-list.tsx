'use client';

import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useEffect, useRef, useState, type PointerEvent } from 'react';
import { ChevronDown, ChevronUp, Film, GripVertical, Pin } from 'lucide-react';
import {
  listCategories,
  listContents,
  moveContent,
  setContentPinned,
  type Category,
  type ContentSummary,
} from '@/lib/api';
import { afterIdForDrop, afterIdForStep, reorder, withPinned } from './library-order';
import { CategoryBar } from './category-bar';

const dateFormat = new Intl.DateTimeFormat('th-TH', { dateStyle: 'medium', timeStyle: 'short' });

export function ContentList() {
  const params = useSearchParams();
  const router = useRouter();
  const filter = params.get('category') ?? '';
  const [items, setItems] = useState<ContentSummary[] | null>(null);
  const [categories, setCategories] = useState<Category[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [categoriesFailed, setCategoriesFailed] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  const [loadedFilter, setLoadedFilter] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState('');
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [over, setOver] = useState<{ id: string; placeAfter: boolean } | null>(null);

  const filterRef = useRef(filter);
  const overRef = useRef(over);
  const dragRef = useRef<{ id: string; pinned: boolean; x: number; y: number; active: boolean } | null>(
    null,
  );

  useEffect(() => {
    filterRef.current = filter;
  }, [filter]);

  useEffect(() => {
    const controller = new AbortController();
    listContents(controller.signal, filter || undefined)
      .then((next) => {
        if (controller.signal.aborted) return;
        setItems(next);
        setLoadedFilter(filter);
        setFailed(false);
      })
      .catch(() => {
        if (!controller.signal.aborted) setFailed(true);
      });
    return () => controller.abort();
  }, [filter, reloadKey]);

  useEffect(() => {
    const controller = new AbortController();
    listCategories(controller.signal)
      .then((next) => {
        if (controller.signal.aborted) return;
        setCategories(next);
        setCategoriesFailed(false);
      })
      .catch(() => {
        if (!controller.signal.aborted) setCategoriesFailed(true);
      });
    return () => controller.abort();
  }, [reloadKey]);

  function setFilter(value: string) {
    const next = new URLSearchParams(params.toString());
    if (value) next.set('category', value);
    else next.delete('category');
    const query = next.toString();
    router.replace(query ? `/contents/?${query}` : '/contents/');
  }

  async function commit(next: ContentSummary[], request: () => Promise<void>, failure: string) {
    const started = filter;
    const previous = items;
    setItems(next);
    setPending(true);
    setMessage('');
    try {
      await request();
      const fresh = await listContents(undefined, started || undefined);
      if (filterRef.current !== started) return;
      setItems(fresh);
      setLoadedFilter(started);
    } catch {
      if (filterRef.current !== started) return;
      setMessage(failure);
      try {
        const fresh = await listContents(undefined, started || undefined);
        if (filterRef.current === started) setItems(fresh);
      } catch {
        if (filterRef.current === started) setItems(previous);
      }
    } finally {
      setPending(false);
    }
  }

  function togglePin(item: ContentSummary) {
    if (!items || pending) return;
    const pinned = !item.pinned;
    void commit(
      withPinned(items, item.id, pinned),
      () => setContentPinned(item.id, pinned),
      pinned ? 'ปักหมุดไม่สำเร็จ' : 'เลิกปักหมุดไม่สำเร็จ',
    );
  }

  function step(item: ContentSummary, direction: -1 | 1) {
    if (!items || pending) return;
    const afterId = afterIdForStep(items, item.id, direction);
    if (afterId === undefined) return;
    void commit(reorder(items, item.id, afterId), () => moveContent(item.id, afterId), 'จัดลำดับไม่สำเร็จ');
  }

  function onGripDown(event: PointerEvent<HTMLButtonElement>, item: ContentSummary) {
    if (event.button !== 0 || pending) return;
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    dragRef.current = { id: item.id, pinned: item.pinned, x: event.clientX, y: event.clientY, active: false };
  }

  function onGripMove(event: PointerEvent<HTMLButtonElement>) {
    const drag = dragRef.current;
    if (!drag) return;
    if (!drag.active && Math.hypot(event.clientX - drag.x, event.clientY - drag.y) < 6) return;
    drag.active = true;
    setDraggingId(drag.id);
    if (event.clientY < 72) window.scrollBy(0, -16);
    else if (event.clientY > window.innerHeight - 72) window.scrollBy(0, 16);
    const node = document.elementFromPoint(event.clientX, event.clientY);
    const card = node?.closest<HTMLElement>('[data-content-id]');
    const targetId = card?.dataset.contentId;
    const target = items?.find((entry) => entry.id === targetId);
    if (!card || !target || target.id === drag.id || target.pinned !== drag.pinned) {
      overRef.current = null;
      setOver(null);
      return;
    }
    const rect = card.getBoundingClientRect();
    const score =
      (event.clientX - rect.left) / rect.width + (event.clientY - rect.top) / rect.height;
    const next = { id: target.id, placeAfter: score >= 1 };
    overRef.current = next;
    setOver(next);
  }

  function finishDrag() {
    const drag = dragRef.current;
    const target = overRef.current;
    dragRef.current = null;
    overRef.current = null;
    setDraggingId(null);
    setOver(null);
    if (!drag?.active || !target || !items) return;
    const afterId = afterIdForDrop(items, drag.id, target.id, target.placeAfter);
    if (afterId === undefined) return;
    void commit(reorder(items, drag.id, afterId), () => moveContent(drag.id, afterId), 'จัดลำดับไม่สำเร็จ');
  }

  const selected = categories?.find((category) => category.id === filter);
  const filterLabel = filter === 'uncategorized' ? 'ไม่มีหมวดหมู่' : selected ? selected.name : '';
  const ready = items && loadedFilter === filter ? items : null;

  return (
    <>
      {categories && (
        <CategoryBar
          categories={categories}
          filter={filter}
          onFilter={setFilter}
          onReload={() => setReloadKey((key) => key + 1)}
        />
      )}
      {categoriesFailed && !failed && <p className="mt-5 text-sm text-muted">โหลดหมวดหมู่ไม่ได้</p>}
      {failed ? (
        <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
          <h2 className="font-medium">โหลดเนื้อหาไม่ได้</h2>
          <p className="mt-2 text-sm leading-6 text-muted">
            ตรวจว่า API เปิดอยู่และเชื่อมฐานข้อมูลแล้ว จากนั้นรีเฟรชหน้านี้
          </p>
        </section>
      ) : !ready ? (
        <p className="mt-5 text-sm text-muted">กำลังโหลดเนื้อหา…</p>
      ) : ready.length === 0 ? (
        <Empty filter={filter} filterLabel={filterLabel} />
      ) : (
        <>
          <p className="mt-5 text-sm text-muted">
            {ready.length.toLocaleString('th-TH')} เนื้อหา
            {filterLabel ? ` · ${filterLabel}` : ''}
          </p>
          <p className="mt-1 text-sm text-muted">ปักหมุดอยู่บนสุด ลากไอคอนหรือใช้ปุ่มลูกศรเพื่อจัดลำดับ</p>
          {message && (
            <p role="status" className="mt-3 text-sm text-red-300">
              {message}
            </p>
          )}
          <LibraryGroup
            title={ready.some((item) => item.pinned) ? 'ปักหมุด' : ''}
            items={ready.filter((item) => item.pinned)}
            all={ready}
            pending={pending}
            draggingId={draggingId}
            over={over}
            onPin={togglePin}
            onStep={step}
            onGripDown={onGripDown}
            onGripMove={onGripMove}
            onGripUp={finishDrag}
          />
          <LibraryGroup
            title={ready.some((item) => item.pinned) && ready.some((item) => !item.pinned) ? 'เนื้อหาอื่น' : ''}
            items={ready.filter((item) => !item.pinned)}
            all={ready}
            pending={pending}
            draggingId={draggingId}
            over={over}
            onPin={togglePin}
            onStep={step}
            onGripDown={onGripDown}
            onGripMove={onGripMove}
            onGripUp={finishDrag}
          />
        </>
      )}
    </>
  );
}

function LibraryGroup({
  title,
  items,
  all,
  pending,
  draggingId,
  over,
  onPin,
  onStep,
  onGripDown,
  onGripMove,
  onGripUp,
}: {
  title: string;
  items: ContentSummary[];
  all: ContentSummary[];
  pending: boolean;
  draggingId: string | null;
  over: { id: string; placeAfter: boolean } | null;
  onPin: (item: ContentSummary) => void;
  onStep: (item: ContentSummary, direction: -1 | 1) => void;
  onGripDown: (event: PointerEvent<HTMLButtonElement>, item: ContentSummary) => void;
  onGripMove: (event: PointerEvent<HTMLButtonElement>) => void;
  onGripUp: () => void;
}) {
  if (items.length === 0) return null;
  return (
    <section className={title === 'เนื้อหาอื่น' ? 'mt-8' : 'mt-5'}>
      {title && (
        <h2 className={`mb-3 text-sm font-medium ${title === 'ปักหมุด' ? 'text-brand' : 'text-muted'}`}>
          {title}
        </h2>
      )}
      <ul className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
        {items.map((item) => {
          const name = item.title || 'ไม่มีหัวข้อ';
          const canUp = afterIdForStep(all, item.id, -1) !== undefined;
          const canDown = afterIdForStep(all, item.id, 1) !== undefined;
          const drop = over?.id === item.id;
          return (
            <li
              key={item.id}
              data-content-id={item.id}
              className={`flex flex-col overflow-hidden rounded-2xl border bg-surface ${
                item.pinned ? 'border-brand/50' : 'border-edge'
              } ${draggingId === item.id ? 'pointer-events-none opacity-60' : ''} ${
                drop ? 'ring-2 ring-brand' : ''
              }`}
            >
              <Link
                href={`/contents/editor/?id=${item.id}`}
                className="group flex flex-1 flex-col overflow-hidden"
              >
                <div className="relative aspect-video overflow-hidden bg-black">
                  {item.coverUrl ? (
                    // eslint-disable-next-line @next/next/no-img-element -- presigned R2 URL, static export
                    <img
                      src={item.coverUrl}
                      alt=""
                      loading="lazy"
                      style={{
                        objectPosition: `${item.coverPositionX}% ${item.coverPositionY}%`,
                      }}
                      className="size-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
                    />
                  ) : (
                    <div className="size-full bg-gradient-to-br from-elevated to-canvas p-5">
                      <p className="line-clamp-5 text-xs leading-6 text-muted/70">
                        {item.excerpt || item.title || 'ไม่มีข้อความ'}
                      </p>
                    </div>
                  )}
                  {item.pinned && (
                    <span className="absolute top-3 left-3 rounded-full bg-brand px-2.5 py-1 text-[11px] font-medium text-canvas">
                      ปักหมุด
                    </span>
                  )}
                  {item.videoCount > 0 && (
                    <span className="absolute right-3 bottom-3 flex items-center gap-1 rounded-full bg-black/70 px-2.5 py-1 text-[11px] text-ink backdrop-blur">
                      <Film size={12} aria-hidden /> {item.videoCount} วิดีโอ
                    </span>
                  )}
                </div>
                <div className="flex flex-1 flex-col p-4">
                  {item.categoryName && (
                    <p className="mb-1 truncate text-[11px] font-medium text-brand">{item.categoryName}</p>
                  )}
                  <h3 className="line-clamp-2 font-medium leading-snug group-hover:text-brand">{name}</h3>
                  {item.excerpt && (
                    <p className="mt-2 line-clamp-2 text-sm leading-6 text-muted">{item.excerpt}</p>
                  )}
                  <p className="mt-auto truncate pt-3 text-xs text-muted">
                    {item.authorName} · แก้ไข {dateFormat.format(new Date(item.updatedAt))}
                  </p>
                </div>
              </Link>
              <div className="flex items-center gap-1 border-t border-edge px-2 py-1">
                <button
                  type="button"
                  aria-label={`ลากเพื่อจัดลำดับ ${name}`}
                  disabled={pending}
                  onPointerDown={(event) => onGripDown(event, item)}
                  onPointerMove={onGripMove}
                  onPointerUp={onGripUp}
                  onPointerCancel={onGripUp}
                  onContextMenu={(event) => event.preventDefault()}
                  style={{ touchAction: 'none' }}
                  className="grid size-11 place-items-center rounded-lg text-muted hover:bg-elevated hover:text-ink disabled:opacity-40"
                >
                  <GripVertical size={18} aria-hidden />
                </button>
                <button
                  type="button"
                  aria-pressed={item.pinned}
                  aria-label={item.pinned ? `เลิกปักหมุด ${name}` : `ปักหมุด ${name}`}
                  disabled={pending}
                  onClick={() => onPin(item)}
                  className={`grid size-11 place-items-center rounded-lg hover:bg-elevated disabled:opacity-40 ${
                    item.pinned ? 'text-brand' : 'text-muted hover:text-ink'
                  }`}
                >
                  <Pin size={18} aria-hidden className={item.pinned ? 'fill-current' : ''} />
                </button>
                <span className="ml-auto flex">
                  <button
                    type="button"
                    aria-label={`เลื่อนขึ้น ${name}`}
                    disabled={pending || !canUp}
                    onClick={() => onStep(item, -1)}
                    className="grid size-11 place-items-center rounded-lg text-muted hover:bg-elevated hover:text-ink disabled:opacity-40"
                  >
                    <ChevronUp size={18} aria-hidden />
                  </button>
                  <button
                    type="button"
                    aria-label={`เลื่อนลง ${name}`}
                    disabled={pending || !canDown}
                    onClick={() => onStep(item, 1)}
                    className="grid size-11 place-items-center rounded-lg text-muted hover:bg-elevated hover:text-ink disabled:opacity-40"
                  >
                    <ChevronDown size={18} aria-hidden />
                  </button>
                </span>
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

function Empty({ filter, filterLabel }: { filter: string; filterLabel: string }) {
  if (!filter) {
    return (
      <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
        <h2 className="font-medium">ยังไม่มีเนื้อหา</h2>
        <p className="mt-2 text-sm leading-6 text-muted">กด “เขียนเนื้อหา” เพื่อเปิดพื้นที่เขียน</p>
      </section>
    );
  }
  const title = filter === 'uncategorized' ? 'ไม่มีเนื้อหาที่ไม่มีหมวดหมู่' : 'ไม่มีเนื้อหาในหมวดนี้';
  const body =
    filter === 'uncategorized'
      ? 'เนื้อหาทุกชิ้นอยู่ในหมวดหมู่แล้ว'
      : filterLabel
        ? `หมวด “${filterLabel}” ยังว่าง`
        : 'หมวดหมู่นี้อาจถูกลบไปแล้ว';
  return (
    <section className="mt-5 rounded-2xl border border-edge bg-surface px-6 py-16 text-center">
      <h2 className="font-medium">{title}</h2>
      <p className="mt-2 text-sm leading-6 text-muted">{body}</p>
    </section>
  );
}
