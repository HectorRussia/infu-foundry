// Pure confirm/alert state. The dialog component and the node:test file both use this
// so focus, dismiss, and queue behaviour stay out of the React tree.

export type DialogKind = 'confirm' | 'alert';
export type DialogPhase = 'open' | 'pending' | 'closing';
export type FocusTarget = 'cancel' | 'confirm';

export const CANCEL_LABEL = 'ยกเลิก';
// No screen has an acknowledge button yet. Callers can pass a label already used
// on the page; this is only the fallback for a notice.
export const ALERT_ACK_LABEL = 'ตกลง';
export const PENDING_LABEL = 'กำลังทำ…';
export const CLOSE_MS = 200;

export type DialogRequest = {
  kind: DialogKind;
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel: string;
  danger: boolean;
};

export type ConfirmInput = {
  title: string;
  message: string;
  confirmLabel: string;
  cancelLabel?: string;
  /** Irreversible actions use the danger style and open with focus on ยกเลิก. */
  danger?: boolean;
};

export type AlertInput = {
  title: string;
  message: string;
  confirmLabel?: string;
};

export type QueueState<T> = { active: T | null; waiting: T[] };

export function requestFromConfirm(input: ConfirmInput): DialogRequest {
  return {
    kind: 'confirm',
    title: input.title,
    message: input.message,
    confirmLabel: input.confirmLabel,
    cancelLabel: input.cancelLabel ?? CANCEL_LABEL,
    danger: input.danger ?? false,
  };
}

export function requestFromAlert(input: AlertInput): DialogRequest {
  return {
    kind: 'alert',
    title: input.title,
    message: input.message,
    confirmLabel: input.confirmLabel ?? ALERT_ACK_LABEL,
    cancelLabel: '',
    danger: false,
  };
}

/** Danger confirms land on ยกเลิก. Notices and reversible confirms land on the action. */
export function initialFocusTarget(request: DialogRequest): FocusTarget {
  if (request.kind === 'confirm' && request.danger) return 'cancel';
  return 'confirm';
}

/** Esc and the backdrop cancel only while the user can still change their mind. */
export function canDismiss(phase: DialogPhase): boolean {
  return phase === 'open';
}

export function confirmButtonLabel(request: DialogRequest, phase: DialogPhase): string {
  return phase === 'pending' ? PENDING_LABEL : request.confirmLabel;
}

export function closeDelayMs(reducedMotion: boolean): number {
  return reducedMotion ? 0 : CLOSE_MS;
}

export function enqueue<T>(state: QueueState<T>, item: T): QueueState<T> {
  if (state.active) return { active: state.active, waiting: [...state.waiting, item] };
  return { active: item, waiting: state.waiting };
}

export function settleQueue<T>(state: QueueState<T>): QueueState<T> {
  const [next, ...rest] = state.waiting;
  return { active: next ?? null, waiting: rest };
}
