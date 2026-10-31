'use client';

import {
  createContext,
  useCallback,
  useContext,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactNode,
} from 'react';
import {
  canDismiss,
  closeDelayMs,
  confirmButtonLabel,
  beginRequest,
  initialFocusTarget,
  requestFromAlert,
  requestFromConfirm,
  settleQueue,
  type AlertInput,
  type ConfirmInput,
  type DialogPhase,
  type DialogRequest,
  type QueueState,
} from './confirm-request';

const FOCUSABLE =
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

export type ConfirmOptions = ConfirmInput & {
  /** The dialog stays open and the confirm button shows กำลังทำ… until this settles. */
  onConfirm?: () => Promise<void> | void;
};

type Task = {
  id: number;
  request: DialogRequest;
  onConfirm?: () => Promise<void> | void;
  trigger: HTMLElement | null;
  finish: (result: 'confirm' | 'cancel', error?: unknown) => void;
};

type DialogApi = {
  confirm: (options: ConfirmOptions) => Promise<boolean>;
  notice: (options: AlertInput) => Promise<void>;
};

const ConfirmContext = createContext<DialogApi | null>(null);

function useDialogApi() {
  const api = useContext(ConfirmContext);
  if (!api) throw new Error('useConfirm must be used within ConfirmProvider');
  return api;
}

export function useConfirm() {
  return useDialogApi().confirm;
}

export function useAlert() {
  return useDialogApi().notice;
}

function activeTrigger(): HTMLElement | null {
  const el = document.activeElement;
  return el instanceof HTMLElement ? el : null;
}

let nextId = 0;

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [queue, setQueue] = useState<QueueState<Task>>({ active: null, waiting: [] });
  // Updated inside push/settle so a request made while an action is resolving is not dropped
  // before React re-renders.
  const queueRef = useRef(queue);

  const push = useCallback((task: Task) => {
    const next = beginRequest(queueRef.current, task);
    if (!next) return false;
    queueRef.current = next;
    setQueue(next);
    return true;
  }, []);

  const settle = useCallback((task: Task, result: 'confirm' | 'cancel', error?: unknown) => {
    const current = queueRef.current;
    const next = current.active?.id === task.id ? settleQueue(current) : current;
    queueRef.current = next;
    task.finish(result, error);
    setQueue(next);
    if (next.active || !task.trigger?.isConnected) return;
    const trigger = task.trigger;
    trigger.focus();
    // showModal() also restores focus, sometimes after this turn. Put it back on the trigger.
    requestAnimationFrame(() => {
      if (trigger.isConnected) trigger.focus();
    });
  }, []);

  const api = useMemo<DialogApi>(
    () => ({
      confirm(options) {
        const trigger = activeTrigger();
        let resolve!: (value: boolean) => void;
        let reject!: (reason: unknown) => void;
        const promise = new Promise<boolean>((res, rej) => {
          resolve = res;
          reject = rej;
        });
        // Queue state is updated synchronously, so a second click in this frame does not stack.
        const accepted = push({
          id: ++nextId,
          request: requestFromConfirm(options),
          onConfirm: options.onConfirm,
          trigger,
          finish(result, error) {
            if (error) reject(error);
            else resolve(result === 'confirm');
          },
        });
        if (!accepted) resolve(false);
        return promise;
      },
      notice(options) {
        const trigger = activeTrigger();
        let resolve!: () => void;
        let reject!: (reason: unknown) => void;
        const promise = new Promise<void>((res, rej) => {
          resolve = res;
          reject = rej;
        });
        const accepted = push({
          id: ++nextId,
          request: requestFromAlert(options),
          trigger,
          finish(_result, error) {
            if (error) reject(error);
            else resolve();
          },
        });
        if (!accepted) resolve();
        return promise;
      },
    }),
    [push],
  );

  return (
    <ConfirmContext.Provider value={api}>
      {children}
      {queue.active && (
        <ConfirmDialog key={queue.active.id} task={queue.active} onSettled={settle} />
      )}
    </ConfirmContext.Provider>
  );
}

function ConfirmDialog({
  task,
  onSettled,
}: {
  task: Task;
  onSettled: (task: Task, result: 'confirm' | 'cancel', error?: unknown) => void;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);
  const confirmRef = useRef<HTMLButtonElement>(null);
  const phaseRef = useRef<DialogPhase>('open');
  const settled = useRef(false);
  const [phase, setPhase] = useState<DialogPhase>('open');
  const [busy, setBusy] = useState(false);
  const [reduced] = useState(
    () =>
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches,
  );
  const [presented, setPresented] = useState(reduced);
  const titleId = `confirm-title-${task.id}`;
  const messageId = `confirm-message-${task.id}`;
  const { request } = task;
  const focus = initialFocusTarget(request);
  const locked = phase !== 'open';
  const hidden = !presented || phase === 'closing';

  function setPhaseBoth(next: DialogPhase) {
    phaseRef.current = next;
    setPhase(next);
  }

  const dismiss = useCallback(
    async (result: 'confirm' | 'cancel', error?: unknown) => {
      if (settled.current) return;
      if (result === 'cancel' && !canDismiss(phaseRef.current)) return;
      if (phaseRef.current === 'closing') return;
      settled.current = true;
      setPhaseBoth('closing');
      const delay = closeDelayMs(reduced);
      if (delay > 0) await new Promise((resolve) => setTimeout(resolve, delay));
      dialogRef.current?.close();
      onSettled(task, result, error);
    },
    [onSettled, reduced, task],
  );

  useLayoutEffect(() => {
    const previous = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = previous;
    };
  }, []);

  useLayoutEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (!dialog.open) dialog.showModal();
    (focus === 'cancel' ? cancelRef : confirmRef).current?.focus();
    const onCancel = (event: Event) => {
      event.preventDefault();
      if (canDismiss(phaseRef.current)) void dismiss('cancel');
    };
    dialog.addEventListener('cancel', onCancel);
    return () => dialog.removeEventListener('cancel', onCancel);
  }, [dismiss, focus]);

  useLayoutEffect(() => {
    if (reduced) return;
    const id = requestAnimationFrame(() => setPresented(true));
    return () => cancelAnimationFrame(id);
  }, [reduced]);

  async function accept() {
    if (!canDismiss(phaseRef.current)) return;
    if (!task.onConfirm) {
      await dismiss('confirm');
      return;
    }
    setBusy(true);
    setPhaseBoth('pending');
    dialogRef.current?.focus();
    try {
      await task.onConfirm();
      await dismiss('confirm');
    } catch (error) {
      await dismiss('confirm', error);
    }
  }

  function trapTab(event: KeyboardEvent<HTMLDialogElement>) {
    if (event.key !== 'Tab') return;
    const dialog = dialogRef.current;
    if (!dialog) return;
    const nodes = [...dialog.querySelectorAll<HTMLElement>(FOCUSABLE)];
    if (nodes.length === 0) {
      event.preventDefault();
      dialog.focus();
      return;
    }
    const first = nodes[0];
    const last = nodes[nodes.length - 1];
    const active = document.activeElement;
    if (event.shiftKey && (active === first || !dialog.contains(active))) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && (active === last || !dialog.contains(active))) {
      event.preventDefault();
      first.focus();
    }
  }

  const button =
    'inline-flex min-h-12 w-full items-center justify-center rounded-xl px-4 text-sm font-semibold transition-colors disabled:opacity-50 sm:w-auto sm:min-w-36';

  return (
    <dialog
      ref={dialogRef}
      role="alertdialog"
      aria-modal="true"
      aria-labelledby={titleId}
      aria-describedby={messageId}
      aria-busy={busy || undefined}
      tabIndex={-1}
      onKeyDown={trapTab}
      onClick={(event) => {
        if (!canDismiss(phaseRef.current)) return;
        if (panelRef.current?.contains(event.target as Node)) return;
        void dismiss('cancel');
      }}
      className="app-dialog text-ink"
    >
      <div className="relative flex h-full w-full items-end justify-center sm:items-center">
        <div
          aria-hidden
          className={`absolute inset-0 bg-black/60 motion-safe:transition-opacity motion-safe:duration-200 ${hidden ? 'opacity-0' : 'opacity-100'}`}
        />
        <div
          ref={panelRef}
          className={`relative z-10 max-h-[85dvh] w-full overflow-y-auto rounded-t-2xl border border-edge bg-elevated px-5 pt-3 pb-[max(1.25rem,env(safe-area-inset-bottom))] shadow-2xl motion-safe:transition-[transform,opacity] motion-safe:duration-200 motion-safe:ease-out sm:w-[min(100%-2rem,28rem)] sm:rounded-2xl sm:px-6 sm:py-6 ${
            !reduced && hidden
              ? 'translate-y-full sm:translate-y-4 sm:opacity-0'
              : 'translate-y-0 opacity-100'
          }`}
        >
          <span
            aria-hidden
            className="mx-auto mb-3 block h-1 w-10 rounded-full bg-edge sm:hidden"
          />
          <h2 id={titleId} className="text-base font-semibold text-ink">
            {request.title}
          </h2>
          <p
            id={messageId}
            className="mt-2 text-sm leading-6 break-words whitespace-pre-line text-muted"
          >
            {request.message}
          </p>
          <div className="mt-5 flex flex-col gap-2 sm:flex-row sm:justify-end">
            {request.kind === 'confirm' && (
              <button
                ref={cancelRef}
                type="button"
                disabled={locked}
                onClick={() => void dismiss('cancel')}
                className={`${button} border border-edge bg-surface font-medium text-ink hover:bg-canvas`}
              >
                {request.cancelLabel}
              </button>
            )}
            <button
              ref={confirmRef}
              type="button"
              disabled={locked}
              aria-busy={busy || undefined}
              onClick={() => void accept()}
              className={
                request.danger
                  ? `${button} border border-red-400/40 bg-red-400/15 text-red-300 hover:bg-red-400/25`
                  : `${button} bg-brand text-canvas hover:bg-brand-hover`
              }
            >
              {confirmButtonLabel(request, busy ? 'pending' : 'open')}
            </button>
          </div>
        </div>
      </div>
    </dialog>
  );
}
