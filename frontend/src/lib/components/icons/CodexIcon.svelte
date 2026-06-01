<script lang="ts">
  /** Codex (OpenAI) icon */
  let { size = 24 }: { size?: number } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
  <rect width="24" height="24" rx="6" fill="#10a37f"/>
  <path d="M12 5.5c-3.584 0-6.5 2.916-6.5 6.5 0 1.686.64 3.23 1.693 4.405l-1.078 2.476a.5.5 0 00.682.628l2.783-1.35A6.458 6.458 0 0012 18.5c3.584 0 6.5-2.916 6.5-6.5S15.584 5.5 12 5.5z" fill="white" fill-opacity="0.9"/>
  <path d="M15.5 10.5l-4.5 3-2-1.5" stroke="#10a37f" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
  <circle cx="14" cy="9" r="1.5" fill="#10a37f"/>
</svg>
