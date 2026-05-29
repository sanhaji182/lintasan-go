import { writable } from 'svelte/store';
import { browser } from '$app/environment';

function createThemeStore() {
  const stored = browser ? localStorage.getItem('lintasan-theme') || 'light' : 'light';
  const { subscribe, set, update } = writable<'light' | 'dark'>(stored as 'light' | 'dark');

  return {
    subscribe,
    toggle: () => {
      update(current => {
        const next = current === 'light' ? 'dark' : 'light';
        if (browser) {
          localStorage.setItem('lintasan-theme', next);
          document.documentElement.setAttribute('data-theme', next);
        }
        return next;
      });
    },
    init: () => {
      if (browser) {
        document.documentElement.setAttribute('data-theme', stored);
      }
    }
  };
}

export const theme = createThemeStore();
