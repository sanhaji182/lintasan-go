<script lang="ts">
  let { status }: { status: string } = $props();

  const map: Record<string, { bg: string; fg: string }> = {
    active: { bg: 'var(--color-success-light)', fg: 'var(--color-success)' },
    success: { bg: 'var(--color-success-light)', fg: 'var(--color-success)' },
    healthy: { bg: 'var(--color-success-light)', fg: 'var(--color-success)' },
    hit: { bg: 'var(--color-info-light)', fg: 'var(--color-info)' },
    cached: { bg: 'var(--color-info-light)', fg: 'var(--color-info)' },
    error: { bg: 'var(--color-error-light)', fg: 'var(--color-error)' },
    failed: { bg: 'var(--color-error-light)', fg: 'var(--color-error)' },
    inactive: { bg: 'var(--color-error-light)', fg: 'var(--color-error)' },
    warning: { bg: 'var(--color-warning-light)', fg: 'var(--color-warning)' },
    pending: { bg: 'var(--color-warning-light)', fg: 'var(--color-warning)' },
  };

  const style = $derived(map[status.toLowerCase()] || { bg: 'var(--color-border-light)', fg: 'var(--color-fg-2)' });
</script>

<span
  class="badge"
  style="background: {style.bg}; color: {style.fg};"
>{status}</span>
