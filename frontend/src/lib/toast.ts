import { writable } from 'svelte/store';

interface ToastItem {
  id: number;
  message: string;
  type: 'success' | 'error' | 'info';
}

let nextId = 0;
const { subscribe, update } = writable<ToastItem[]>([]);

export const toasts = { subscribe };

export function showToast(message: string, type: 'success' | 'error' | 'info' = 'info', duration = 3000) {
  const id = nextId++;
  update(t => [...t, { id, message, type }]);
  setTimeout(() => {
    update(t => t.filter(item => item.id !== id));
  }, duration);
}
