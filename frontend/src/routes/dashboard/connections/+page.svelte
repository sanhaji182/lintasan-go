<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import Spinner from '$lib/components/Spinner.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import { showToast } from '$lib/toast';
  import { Link2, Plus, TestTube2, RefreshCw, Trash2, ToggleLeft, ToggleRight, ChevronDown, X } from 'lucide-svelte';

  let connections = $state<any[]>([]);
  let loading = $state(true);
  let error = $state('');
  let showForm = $state(false);
  let testing = $state<string | null>(null);
  let syncing = $state<string | null>(null);

  let form = $state({ name: '', base_url: '', api_key: '', format: 'openai', priority: 1 });

  const summary = $derived({
    total: connections.length,
    active: connections.filter(c => c.is_active).length,
    formats: [...new Set(connections.map(c => c.format))].length
  });

  onMount(async () => {
    try {
      const res = await api.get<any>('/api/connections');
      connections = res.data || [];
    } catch (e: any) { error = e.message; }
    finally { loading = false; }
  });

  async function toggleActive(conn: any) {
    try {
      await api.patch('/api/connections', { id: conn.id, is_active: !conn.is_active });
      connections = connections.map(c => c.id === conn.id ? { ...c, is_active: !c.is_active } : c);
    } catch (e: any) { error = e.message; }
  }

  async function testConn(id: string) {
    testing = id;
    try {
      const res = await api.post<any>('/api/connections/test', { id });
      showToast(res.success ? 'Connection OK!' : 'Test failed: ' + (res.error || 'unknown'), res.success ? 'success' : 'error');
    } catch (e: any) { showToast('Test failed: ' + e.message, 'error'); }
    finally { testing = null; }
  }

  async function syncModels(id: string) {
    syncing = id;
    try {
      await api.post('/api/models/sync/' + id);
      const res = await api.get<any>('/api/connections');
      connections = res.data || [];
    } catch (e: any) { error = e.message; }
    finally { syncing = null; }
  }

  async function deleteConn(id: string) {
    if (!confirm('Delete this connection?')) return;
    try {
      await api.delete('/api/connections/' + id);
      connections = connections.filter(c => c.id !== id);
    } catch (e: any) { error = e.message; }
  }

  async function createConn() {
    try {
      const res = await api.post<any>('/api/connections', form);
      const updated = await api.get<any>('/api/connections');
      connections = updated.data || [];
      showForm = false;
      form = { name: '', base_url: '', api_key: '', format: 'openai', priority: 1 };
    } catch (e: any) { error = e.message; }
  }
</script>

<div style="animation: fadeInUp 0.4s ease-out;">
  <!-- Summary strip -->
  <div class="card mb-5" style="padding: 0; overflow: hidden;">
    <div class="grid grid-cols-3" style="gap: 1px; background: var(--color-border);">
      {#each [
        { label: 'TOTAL', value: summary.total },
        { label: 'ACTIVE', value: summary.active },
        { label: 'FORMATS', value: summary.formats }
      ] as stat}
        <div class="text-center" style="padding: 16px 20px; background: var(--color-bg-card);">
          <div style="font-size: 11px; font-weight: 500; color: var(--color-fg-3); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 4px;">{stat.label}</div>
          <div style="font-size: 20px; font-weight: 700; color: var(--color-fg-0); font-family: var(--font-mono);">{stat.value}</div>
        </div>
      {/each}
    </div>
  </div>

  <!-- Actions -->
  <div class="flex items-center justify-between mb-5">
    <h2 style="font-size: 18px; font-weight: 600; color: var(--color-fg-0);">Connections</h2>
    <button class="btn-primary flex items-center gap-2" onclick={() => showForm = !showForm}>
      {#if showForm}<X size={16} />{:else}<Plus size={16} />{/if}
      {showForm ? 'Cancel' : 'Add Connection'}
    </button>
  </div>

  <!-- Create form -->
  {#if showForm}
    <div class="card mb-5" style="animation: fadeInScale 0.3s ease-out;">
      <div class="grid grid-cols-1 md:grid-cols-2" style="gap: 12px;">
        <div>
          <label style="font-size: 12px; font-weight: 500; color: var(--color-fg-2); display: block; margin-bottom: 4px;">Name</label>
          <input class="input-field" bind:value={form.name} placeholder="My Provider" />
        </div>
        <div>
          <label style="font-size: 12px; font-weight: 500; color: var(--color-fg-2); display: block; margin-bottom: 4px;">Base URL</label>
          <input class="input-field" bind:value={form.base_url} placeholder="https://api.openai.com/v1" />
        </div>
        <div>
          <label style="font-size: 12px; font-weight: 500; color: var(--color-fg-2); display: block; margin-bottom: 4px;">API Key</label>
          <input class="input-field" type="password" bind:value={form.api_key} placeholder="sk-..." />
        </div>
        <div>
          <label style="font-size: 12px; font-weight: 500; color: var(--color-fg-2); display: block; margin-bottom: 4px;">Format</label>
          <select class="input-field" bind:value={form.format}>
            <option value="openai">OpenAI</option>
            <option value="anthropic">Anthropic</option>
            <option value="gemini">Gemini</option>
          </select>
        </div>
      </div>
      <div class="flex justify-end mt-4">
        <button class="btn-primary" onclick={createConn}>Create Connection</button>
      </div>
    </div>
  {/if}

  {#if loading}
    <Spinner />
  {:else if connections.length === 0}
    <div class="card">
      <EmptyState icon={Link2} title="No connections" description="Add your first LLM provider connection." />
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3" style="gap: 16px;">
      {#each connections as conn}
        <div class="card relative" style="padding: 20px; transition: var(--transition);" onmouseenter={(e) => (e.currentTarget as HTMLElement).style.boxShadow = 'var(--shadow-md)'} onmouseleave={(e) => (e.currentTarget as HTMLElement).style.boxShadow = 'var(--shadow)'}>
          <!-- Status dot -->
          <div class="absolute" style="top: 16px; right: 16px; width: 8px; height: 8px; border-radius: 50%; background: {conn.is_active ? 'var(--color-success)' : 'var(--color-error)'}; box-shadow: 0 0 {conn.is_active ? '8px rgba(16,185,129,0.5)' : '8px rgba(239,68,68,0.5)'};"></div>

          <div class="flex items-center gap-3 mb-3">
            <div class="flex items-center justify-center rounded-lg" style="width: 36px; height: 36px; background: var(--color-primary-light); border-radius: 10px;">
              <Link2 size={18} style="color: var(--color-primary);" />
            </div>
            <div>
              <div style="font-size: 14px; font-weight: 600; color: var(--color-fg-0);">{conn.name}</div>
              <div style="font-size: 11px; color: var(--color-fg-3); font-family: var(--font-mono);">{conn.base_url}</div>
            </div>
          </div>

          <div class="flex flex-wrap gap-2 mb-3">
            <span class="badge" style="background: var(--color-purple-light); color: var(--color-purple);">{conn.format}</span>
            <span class="badge" style="background: var(--color-info-light); color: var(--color-info);">P{conn.priority}</span>
            <span class="badge" style="background: var(--color-border-light); color: var(--color-fg-2);">{conn.models_count || 0} models</span>
          </div>

          {#if conn.api_key}
            <div style="font-size: 11px; color: var(--color-fg-3); font-family: var(--font-mono); margin-bottom: 12px; padding: 4px 8px; background: var(--color-bg-body); border-radius: 4px;">{conn.api_key}</div>
          {/if}

          <div class="flex items-center gap-2">
            <button class="btn-secondary flex items-center gap-1" style="padding: 6px 10px; font-size: 12px;" onclick={() => toggleActive(conn)} title={conn.is_active ? 'Deactivate' : 'Activate'}>
              {#if conn.is_active}<ToggleRight size={16} style="color: var(--color-success);" />{:else}<ToggleLeft size={16} />{/if}
            </button>
            <button class="btn-secondary flex items-center gap-1" style="padding: 6px 10px; font-size: 12px;" onclick={() => testConn(conn.id)} disabled={testing === conn.id}>
              <TestTube2 size={14} /> {testing === conn.id ? '...' : 'Test'}
            </button>
            <button class="btn-secondary flex items-center gap-1" style="padding: 6px 10px; font-size: 12px;" onclick={() => syncModels(conn.id)} disabled={syncing === conn.id}>
              <RefreshCw size={14} class={syncing === conn.id ? 'animate-spin' : ''} /> Sync
            </button>
            <button class="btn-secondary flex items-center gap-1" style="padding: 6px 10px; font-size: 12px; color: var(--color-error);" onclick={() => deleteConn(conn.id)}>
              <Trash2 size={14} />
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
