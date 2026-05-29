<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { Menu, UserCircle2, LogOut, LogIn } from 'lucide-svelte';

  let { title = 'Overview', open = $bindable(false) }: { title?: string; open?: boolean } = $props();

  let isAuthenticated = $state(false);
  let username = $state('');
  let role = $state('');

  function refreshAuthState() {
    if (typeof window === 'undefined') return;
    const token = localStorage.getItem('lintasan_token');
    const userStr = localStorage.getItem('lintasan_user');
    isAuthenticated = !!token;
    if (userStr) {
      try {
        const user = JSON.parse(userStr);
        username = user?.username || '';
        role = user?.role || '';
      } catch {
        username = '';
        role = '';
      }
    } else {
      username = '';
      role = '';
    }
  }

  async function handleLogout() {
    try {
      await fetch('/api/auth/logout', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('lintasan_token') || ''}`
        }
      });
    } catch {
      // ignore network/server errors; client-side logout still proceeds
    }

    localStorage.removeItem('lintasan_token');
    localStorage.removeItem('lintasan_user');
    refreshAuthState();
    await goto('/login');
  }

  onMount(() => {
    refreshAuthState();
    const onStorage = () => refreshAuthState();
    window.addEventListener('storage', onStorage);

    const onFocus = () => refreshAuthState();
    window.addEventListener('focus', onFocus);

    return () => {
      window.removeEventListener('storage', onStorage);
      window.removeEventListener('focus', onFocus);
    };
  });
</script>

<header
  class="sticky top-0 z-40 flex items-center justify-between border-b backdrop-blur-xl"
  style="
    height: var(--header-h);
    padding: 0 24px;
    background: var(--color-bg-card);
    border-color: var(--color-border);
  "
>
  <div class="flex items-center gap-3">
    <button
      class="flex items-center justify-center md:hidden"
      style="width: 44px; height: 44px; border-radius: var(--radius-sm); color: var(--color-fg-1);"
      onclick={() => open = !open}
    >
      <Menu size={20} />
    </button>
    <h1 class="font-semibold" style="font-size: 15px; color: var(--color-fg-0);">{title}</h1>
  </div>

  <div class="flex items-center gap-2">
    <div
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-mono"
      style="background: var(--color-success-light); color: var(--color-success); font-size: 11px;"
    >
      <span class="w-1.5 h-1.5 rounded-full" style="background: var(--color-success); animation: dotPulse 2s infinite;"></span>
      Online
    </div>

    {#if isAuthenticated}
      <a
        href="/dashboard/users"
        class="flex items-center gap-1.5 px-3 py-2 rounded-md"
        style="border: 1px solid var(--color-border); color: var(--color-fg-1); font-size: 12px;"
        title="User Management"
      >
        <UserCircle2 size={14} />
        <span style="display: inline-block; max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
          {username || 'Account'}
        </span>
      </a>

      <button
        class="btn-secondary flex items-center gap-1.5"
        style="padding: 7px 10px;"
        onclick={handleLogout}
        title="Logout"
      >
        <LogOut size={14} />
        Logout
      </button>
    {:else}
      <a
        href="/login"
        class="btn-primary flex items-center gap-1.5"
        style="padding: 7px 10px;"
      >
        <LogIn size={14} />
        Login
      </a>
    {/if}

    <div
      class="flex items-center justify-center rounded-full text-xs font-semibold text-white"
      style="width: 32px; height: 32px; background: linear-gradient(135deg, var(--color-primary) 0%, #6366f1 100%); font-size: 12px;"
      title={role ? `Role: ${role}` : 'Lintasan'}
    >L</div>
  </div>
</header>
