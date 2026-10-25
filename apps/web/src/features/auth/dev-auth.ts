'use client';

import { useSyncExternalStore } from 'react';

// UI preview only. Never use this as API authorization or production authentication.
export const DEV_LOGIN = process.env.NODE_ENV === 'development';
export const DEV_EMAIL = DEV_LOGIN ? 'dev@infu.local' : '';
export const DEV_PASSWORD = DEV_LOGIN ? 'infu-dev-123' : '';
const SESSION_KEY = 'infu-foundry:dev-session';
const SESSION_EVENT = 'infu-foundry:dev-session-change';

export function hasDevSession() {
  if (!DEV_LOGIN) return false;
  try {
    return sessionStorage.getItem(SESSION_KEY) === 'active';
  } catch {
    return false;
  }
}

function subscribe(notify: () => void) {
  window.addEventListener(SESSION_EVENT, notify);
  window.addEventListener('storage', notify);
  return () => {
    window.removeEventListener(SESSION_EVENT, notify);
    window.removeEventListener('storage', notify);
  };
}

export function useDevSession() {
  return useSyncExternalStore(subscribe, hasDevSession, () => false);
}

export function loginDev(email: string, password: string) {
  if (!DEV_LOGIN || email.trim().toLowerCase() !== DEV_EMAIL || password !== DEV_PASSWORD) {
    return 'อีเมลหรือรหัสผ่านไม่ถูกต้อง';
  }
  try {
    sessionStorage.setItem(SESSION_KEY, 'active');
    window.dispatchEvent(new Event(SESSION_EVENT));
    return null;
  } catch {
    return 'ไม่สามารถเปิด session ทดสอบได้ กรุณาอนุญาตให้เบราว์เซอร์เก็บข้อมูลเว็บไซต์';
  }
}

export function logoutDev() {
  if (!DEV_LOGIN) return;
  sessionStorage.removeItem(SESSION_KEY);
  window.dispatchEvent(new Event(SESSION_EVENT));
}
