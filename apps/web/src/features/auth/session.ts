'use client';

import { useEffect, useSyncExternalStore } from 'react';
import { APIError, currentUser, UNAUTHORIZED_EVENT, type User } from '@/lib/api';

// The session itself is an HttpOnly cookie owned by the Go API; this store only
// mirrors "who is signed in" for the UI. It is never an authorization boundary.
export type Session =
  | { status: 'loading' }
  | { status: 'signed-out' }
  | { status: 'error' }
  | { status: 'signed-in'; user: User };

let session: Session = { status: 'loading' };
let loading: Promise<void> | null = null;
const listeners = new Set<() => void>();

function set(next: Session) {
  session = next;
  listeners.forEach((notify) => notify());
}

export function setSignedIn(user: User) {
  set({ status: 'signed-in', user });
}

export function setSignedOut() {
  set({ status: 'signed-out' });
}

function load() {
  loading ??= currentUser()
    .then(setSignedIn)
    .catch((error: unknown) => {
      if (error instanceof APIError && error.status === 401) setSignedOut();
      else set({ status: 'error' });
    })
    .finally(() => {
      loading = null;
    });
  return loading;
}

export function retrySession() {
  set({ status: 'loading' });
  void load();
}

function subscribe(notify: () => void) {
  listeners.add(notify);
  window.addEventListener(UNAUTHORIZED_EVENT, setSignedOut);
  return () => {
    listeners.delete(notify);
    window.removeEventListener(UNAUTHORIZED_EVENT, setSignedOut);
  };
}

const serverSession: Session = { status: 'loading' };

export function useSession(): Session {
  const current = useSyncExternalStore(
    subscribe,
    () => session,
    () => serverSession,
  );
  useEffect(() => {
    if (session.status === 'loading') void load();
  }, []);
  return current;
}
