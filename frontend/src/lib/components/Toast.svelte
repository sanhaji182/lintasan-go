<script lang="ts">
  import { toasts } from '$lib/toast';
  import { CheckCircle2, AlertCircle, Info, X } from 'lucide-svelte';

  const colorMap: Record<string, string> = {
    success: 'var(--color-success)',
    error: 'var(--color-error)',
    info: 'var(--color-info)'
  };
  const bgMap: Record<string, string> = {
    success: 'var(--color-success-light)',
    error: 'var(--color-error-light)',
    info: 'var(--color-info-light)'
  };
</script>

{#if $toasts.length > 0}
  <div class="toast-container">
    {#each $toasts as toast (toast.id)}
      <div
        class="toast"
        style="background: {bgMap[toast.type]}; color: {colorMap[toast.type]}; border: 1px solid {colorMap[toast.type]};"
      >
        {#if toast.type === 'success'}
          <CheckCircle2 size={16} />
        {:else if toast.type === 'error'}
          <AlertCircle size={16} />
        {:else}
          <Info size={16} />
        {/if}
        <span style="flex: 1; font-size: 13px; font-weight: 500;">{toast.message}</span>
      </div>
    {/each}
  </div>
{/if}

<style>
  .toast-container {
    position: fixed;
    top: 16px;
    right: 16px;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-width: 380px;
  }
  .toast {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    border-radius: var(--radius-sm);
    box-shadow: var(--shadow-md);
    animation: slideIn 0.3s ease-out;
  }
  @keyframes slideIn {
    from { transform: translateX(100%); opacity: 0; }
    to { transform: translateX(0); opacity: 1; }
  }
</style>
