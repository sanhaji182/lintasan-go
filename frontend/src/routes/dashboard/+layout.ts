import { redirect } from '@sveltejs/kit';
import type { LayoutLoad } from './$types';

export const ssr = false;

export const load: LayoutLoad = async ({ fetch }) => {
  const token = localStorage.getItem('lintasan_token');

  if (!token) {
    throw redirect(302, '/login');
  }

  try {
    const res = await fetch('/api/auth/me', {
      headers: {
        Authorization: `Bearer ${token}`
      }
    });

    if (!res.ok) {
      throw new Error('unauthorized');
    }
  } catch {
    localStorage.removeItem('lintasan_token');
    localStorage.removeItem('lintasan_user');
    throw redirect(302, '/login');
  }

  return {};
};
