<script lang="ts">
  /** Gemini CLI (Google) icon */
  let { size = 24 }: { size?: number } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
  <rect width="24" height="24" rx="6" fill="white"/>
  <path d="M12 3C7.03 3 3 7.03 3 12s4.03 9 9 9 9-4.03 9-9-4.03-9-9-9z" fill="url(#gemini-grad)"/>
  <defs>
    <linearGradient id="gemini-grad" x1="3" y1="3" x2="21" y2="21">
      <stop offset="0%" stop-color="#4285f4"/>
      <stop offset="25%" stop-color="#ea4335"/>
      <stop offset="50%" stop-color="#fbbc04"/>
      <stop offset="75%" stop-color="#34a853"/>
      <stop offset="100%" stop-color="#4285f4"/>
    </linearGradient>
  </defs>
  <path d="M12 8v4l3 3" stroke="white" stroke-width="1.5" stroke-linecap="round"/>
  <circle cx="12" cy="12" r="2" fill="white"/>
</svg>
