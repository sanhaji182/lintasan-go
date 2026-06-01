<script lang="ts">
  /** GitHub Copilot icon */
  let { size = 24 }: { size?: number } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
  <rect width="24" height="24" rx="6" fill="#6366f1"/>
  <path d="M12 6C8.686 6 6 8.686 6 12c0 2.761 1.79 5.103 4.255 5.865A3.5 3.5 0 0012 18a3.5 3.5 0 001.745-.135C16.21 17.103 18 14.761 18 12c0-3.314-2.686-6-6-6z" fill="white" fill-opacity="0.9"/>
  <ellipse cx="10" cy="11" rx="1.2" ry="1.5" fill="#6366f1"/>
  <ellipse cx="14" cy="11" rx="1.2" ry="1.5" fill="#6366f1"/>
  <path d="M10 14c0 1 .5 2 2 2s2-1 2-2" stroke="#6366f1" stroke-width="0.8" stroke-linecap="round"/>
</svg>
