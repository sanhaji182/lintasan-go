<script lang="ts">
  import { Menu } from 'lucide-svelte';

  let { title = 'Overview', open = $bindable(false) }: { title?: string; open?: boolean } = $props();
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

  <div class="flex items-center gap-3">
    <div
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-mono"
      style="background: var(--color-success-light); color: var(--color-success); font-size: 11px;"
    >
      <span class="w-1.5 h-1.5 rounded-full" style="background: var(--color-success); animation: dotPulse 2s infinite;"></span>
      Online
    </div>
    <div
      class="flex items-center justify-center rounded-full text-xs font-semibold text-white"
      style="width: 32px; height: 32px; background: linear-gradient(135deg, var(--color-primary) 0%, #6366f1 100%); font-size: 12px;"
    >L</div>
  </div>
</header>
