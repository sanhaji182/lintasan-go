<script lang="ts">
  import '../app.css';
  import { theme } from '$lib/stores/theme';
  import { onMount } from 'svelte';

  let { children } = $props();

  onMount(() => theme.init());
</script>

{@render children()}
