<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import Spinner from '$lib/components/Spinner.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import { showToast } from '$lib/toast';
  import {
    GitBranch, GripVertical, Plus, Trash2, Save,
    Server, Tag, Shuffle, RotateCw, CircleDot
  } from 'lucide-svelte';

  interface Combo {
    id: string;
    provider: string;
    strategy: string;
    keys: string[];
    order: number;
  }

  interface Alias {
    id: string;
    alias: string;
    target: string;
  }

  let combos = $state<Combo[]>([]);
  let aliases = $state<Alias[]>([]);
  let loadBalancerStrategy = $state<string>('priority');
  let loading = $state(true);
  let saving = $state(false);
  let error = $state('');
  let draggedIndex = $state<number | null>(null);
  let dragOverIndex = $state<number | null>(null);

  // New alias form
  let showAliasForm = $state(false);
  let newAlias = $state('');
  let newTarget = $state('');

  const strategies = [
    { value: 'round-robin', label: 'Round Robin', icon: RotateCw },
    { value: 'least-latency', label: 'Least Latency', icon: CircleDot },
    { value: 'random', label: 'Random', icon: Shuffle },
  ];

  async function loadCombos() {
    try {
      const res = await api.get<any>('/api/combos');
      const raw = res?.data || res?.combos || [];
      combos = Array.isArray(raw) ? raw.map((c: any, i: number) => ({
        id: c.id || `combo-${i}`,
        provider: c.provider || c.name || 'Unknown',
        strategy: c.strategy || 'priority',
        keys: Array.isArray(c.keys) ? c.keys : [],
        order: c.order ?? i
      })) : [];
    } catch {
      combos = [];
    }
  }

  async function loadAliases() {
    try {
      const res = await api.get<{ data: Record<string, { model?: string } | string> }>('/api/aliases');
      const raw = res.data || {};
      aliases = Object.entries(raw).map(([name, cfg]) => ({
        id: name,
        alias: name,
        target: typeof cfg === 'string' ? cfg : (cfg.model || JSON.stringify(cfg))
      }));
    } catch {
      aliases = [];
    }
  }

  async function loadLoadBalancer() {
    try {
      const res = await api.get<{ data: { strategy: string } }>('/api/load-balancer');
      loadBalancerStrategy = res.data?.strategy || 'priority';
    } catch {
      loadBalancerStrategy = 'priority';
    }
  }

  onMount(async () => {
    loading = true;
    await Promise.all([loadCombos(), loadAliases(), loadLoadBalancer()]);
    loading = false;
  });

  async function updateStrategy(comboId: string, strategy: string) {
    const combo = combos.find(c => c.id === comboId);
    if (combo) combo.strategy = strategy;
    try {
      await api.patch(`/api/routing/combos/${comboId}`, { strategy });
      showToast('Strategy updated', 'success');
    } catch (e: any) {
      error = e.message || 'Failed to update strategy';
      showToast('Failed to update strategy', 'error');
    }
  }

  async function setGlobalStrategy(strategy: string) {
    try {
      await api.put('/api/load-balancer', { strategy });
      loadBalancerStrategy = strategy;
      showToast(`Load balancer set to ${strategy}`, 'success');
    } catch (e: any) {
      error = e.message || 'Failed to update strategy';
      showToast('Failed to update strategy', 'error');
    }
  }

  async function saveOrder() {
    saving = true;
    try {
      const ordered = combos.map((c, i) => ({ id: c.id, order: i }));
      await api.put('/api/routing/combos/reorder', { combos: ordered });
      combos = combos.map((c, i) => ({ ...c, order: i }));
      showToast('Order saved successfully', 'success');
    } catch (e: any) {
      error = e.message || 'Failed to save order';
      showToast('Failed to save order', 'error');
    }
    saving = false;
  }

  function handleDragStart(index: number) {
    draggedIndex = index;
  }

  function handleDragOver(e: DragEvent, index: number) {
    e.preventDefault();
    dragOverIndex = index;
  }

  function handleDragEnd() {
    if (draggedIndex !== null && dragOverIndex !== null && draggedIndex !== dragOverIndex) {
      const items = [...combos];
      const [moved] = items.splice(draggedIndex, 1);
      items.splice(dragOverIndex, 0, moved);
      combos = items.map((c, i) => ({ ...c, order: i }));
    }
    draggedIndex = null;
    dragOverIndex = null;
  }

  async function addAlias() {
    if (!newAlias.trim() || !newTarget.trim()) return;
    try {
      const data = await api.post<{ alias: Alias }>('/api/routing/aliases', {
        alias: newAlias.trim(),
        target: newTarget.trim()
      });
      aliases = [...aliases, data.alias];
      newAlias = '';
      newTarget = '';
      showAliasForm = false;
    } catch (e: any) {
      error = e.message || 'Failed to add alias';
    }
  }

  async function deleteAlias(id: string) {
    try {
      await api.delete(`/api/routing/aliases/${id}`);
      aliases = aliases.filter(a => a.id !== id);
    } catch (e: any) {
      error = e.message || 'Failed to delete alias';
    }
  }
</script>

<div style="display: flex; flex-direction: column; gap: 24px;">
  <!-- Load Balancer Section -->
  <div class="card">
    <div class="flex items-center justify-between" style="margin-bottom: 20px;">
      <div class="flex items-center gap-2.5">
        <div
          class="flex items-center justify-center rounded-xl"
          style="width: 40px; height: 40px; background: var(--color-success-light);"
        >
          <Shuffle size={20} style="color: var(--color-success);" stroke-width={1.8} />
        </div>
        <div>
          <div style="font-size: 15px; font-weight: 600; color: var(--color-fg-0);">Load Balancer</div>
          <div style="font-size: 12px; color: var(--color-fg-3);">Current strategy: <span class="font-mono" style="color: var(--color-primary);">{loadBalancerStrategy}</span></div>
        </div>
      </div>
    </div>
    <div class="flex items-center gap-2 flex-wrap">
      {#each strategies as s}
        <button
          class="badge"
          style="font-size: 12px; padding: 6px 14px; cursor: pointer; border: 1px solid {loadBalancerStrategy === s.value ? 'var(--color-primary)' : 'var(--color-border)'}; background: {loadBalancerStrategy === s.value ? 'var(--color-primary-light)' : 'var(--color-bg-body)'}; color: {loadBalancerStrategy === s.value ? 'var(--color-primary)' : 'var(--color-fg-2)'}; border-radius: var(--radius-sm); transition: var(--transition);"
          onclick={() => setGlobalStrategy(s.value)}
        >
          <s.icon size={14} style="display: inline; vertical-align: -2px; margin-right: 4px;" />
          {s.label}
        </button>
      {/each}
    </div>
  </div>

  <!-- Combos Section -->
  <div class="card">
    <div class="flex items-center justify-between" style="margin-bottom: 20px;">
      <div class="flex items-center gap-2.5">
        <div
          class="flex items-center justify-center rounded-xl"
          style="width: 40px; height: 40px; background: var(--color-primary-light);"
        >
          <GitBranch size={20} style="color: var(--color-primary);" stroke-width={1.8} />
        </div>
        <div>
          <div style="font-size: 15px; font-weight: 600; color: var(--color-fg-0);">Routing Combos</div>
          <div style="font-size: 12px; color: var(--color-fg-3);">Drag to reorder priority. Top combo is tried first.</div>
        </div>
      </div>
      <button class="btn-primary flex items-center gap-1.5" onclick={saveOrder} disabled={saving}>
        <Save size={14} stroke-width={2} />
        {saving ? 'Saving...' : 'Save Order'}
      </button>
    </div>

    {#if loading}
      <Spinner />
    {:else if combos.length === 0}
      <EmptyState
        icon={GitBranch}
        title="No routing combos"
        description="Combos will appear here once configured."
      />
    {:else}
      <div style="display: flex; flex-direction: column; gap: 10px;">
        {#each combos as combo, i (combo.id)}
          <div
            class="combo-card"
            class:drag-over={dragOverIndex === i && draggedIndex !== i}
            class:dragging={draggedIndex === i}
            draggable="true"
            ondragstart={() => handleDragStart(i)}
            ondragover={(e) => handleDragOver(e, i)}
            ondragend={handleDragEnd}
            role="listitem"
          >
            <div class="flex items-center gap-3" style="flex: 1; min-width: 0;">
              <!-- Drag Handle -->
              <div class="drag-handle" title="Drag to reorder">
                <GripVertical size={16} style="color: var(--color-fg-3);" />
              </div>

              <!-- Order Number -->
              <div
                class="flex items-center justify-center rounded-lg font-mono font-bold"
                style="
                  width: 28px; height: 28px; font-size: 12px;
                  background: var(--color-primary-light); color: var(--color-primary);
                  flex-shrink: 0;
                "
              >{i + 1}</div>

              <!-- Combo Info -->
              <div style="flex: 1; min-width: 0;">
                <div class="flex items-center gap-2" style="margin-bottom: 4px;">
                  <div class="flex items-center gap-1.5">
                    <Server size={14} style="color: var(--color-fg-2);" />
                    <span style="font-size: 13px; font-weight: 600; color: var(--color-fg-0);">{combo.provider}</span>
                  </div>
                </div>
                <div class="flex items-center gap-2 flex-wrap">
                  {#each combo.keys as key}
                    <span class="badge" style="background: var(--color-border-light); color: var(--color-fg-2); font-size: 10px;">
                      {key.slice(0, 8)}...
                    </span>
                  {/each}
                </div>
              </div>
            </div>

            <!-- Strategy Selector -->
            <div class="flex items-center gap-2" style="flex-shrink: 0;">
              <select
                class="input-field"
                style="width: 160px; font-size: 12px; padding: 6px 10px;"
                value={combo.strategy}
                onchange={(e) => updateStrategy(combo.id, (e.target as HTMLSelectElement).value)}
              >
                {#each strategies as s}
                  <option value={s.value}>{s.label}</option>
                {/each}
              </select>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Aliases Section -->
  <div class="card">
    <div class="flex items-center justify-between" style="margin-bottom: 20px;">
      <div class="flex items-center gap-2.5">
        <div
          class="flex items-center justify-center rounded-xl"
          style="width: 40px; height: 40px; background: var(--color-purple-light);"
        >
          <Tag size={20} style="color: var(--color-purple);" stroke-width={1.8} />
        </div>
        <div>
          <div style="font-size: 15px; font-weight: 600; color: var(--color-fg-0);">Model Aliases</div>
          <div style="font-size: 12px; color: var(--color-fg-3);">Map friendly names to actual model identifiers.</div>
        </div>
      </div>
      <button
        class="btn-secondary flex items-center gap-1.5"
        onclick={() => showAliasForm = !showAliasForm}
      >
        <Plus size={14} stroke-width={2} />
        Add Alias
      </button>
    </div>

    {#if showAliasForm}
      <div class="alias-form">
        <div class="flex items-center gap-3 flex-wrap">
          <input
            class="input-field"
            style="width: 200px;"
            placeholder="Alias name (e.g. gpt-4)"
            bind:value={newAlias}
          />
          <span style="color: var(--color-fg-3); font-size: 18px; font-weight: 300;">&rarr;</span>
          <input
            class="input-field"
            style="width: 280px;"
            placeholder="Target model (e.g. gpt-4-turbo-preview)"
            bind:value={newTarget}
          />
          <button class="btn-primary" onclick={addAlias}>
            <Plus size={14} stroke-width={2} />
          </button>
          <button class="btn-secondary" onclick={() => { showAliasForm = false; newAlias = ''; newTarget = ''; }}>
            Cancel
          </button>
        </div>
      </div>
    {/if}

    {#if loading}
      <Spinner />
    {:else if aliases.length === 0 && !showAliasForm}
      <EmptyState
        icon={Tag}
        title="No aliases configured"
        description="Create aliases to map short names to full model identifiers."
      />
    {:else}
      <div style="display: flex; flex-direction: column; gap: 8px;">
        {#each aliases as alias (alias.id)}
          <div class="alias-row flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div
                class="flex items-center justify-center rounded-lg"
                style="width: 32px; height: 32px; background: var(--color-purple-light);"
              >
                <Tag size={14} style="color: var(--color-purple);" />
              </div>
              <div>
                <span class="font-mono font-semibold" style="font-size: 13px; color: var(--color-fg-0);">{alias.alias}</span>
                <span style="color: var(--color-fg-3); margin: 0 8px;">&rarr;</span>
                <span class="font-mono" style="font-size: 13px; color: var(--color-fg-2);">{alias.target}</span>
              </div>
            </div>
            <button
              class="btn-icon"
              style="color: var(--color-error);"
              onclick={() => deleteAlias(alias.id)}
              title="Delete alias"
            >
              <Trash2 size={14} />
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  {#if error}
    <div
      class="flex items-center gap-2"
      style="
        padding: 12px 16px; border-radius: var(--radius-sm);
        background: var(--color-error-light); color: var(--color-error);
        font-size: 13px; font-weight: 500;
      "
    >
      {error}
      <button style="margin-left: auto; cursor: pointer; color: var(--color-error); background: none; border: none;" onclick={() => error = ''}>&times;</button>
    </div>
  {/if}
</div>

<style>
  .combo-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 14px 16px;
    background: var(--color-bg-body);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    cursor: grab;
    transition: var(--transition);
  }
  .combo-card:hover {
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px var(--color-primary-glow);
  }
  .combo-card.dragging {
    opacity: 0.5;
    transform: scale(0.98);
  }
  .combo-card.drag-over {
    border-color: var(--color-primary);
    border-style: dashed;
    background: var(--color-primary-light);
  }
  .drag-handle {
    cursor: grab;
    padding: 4px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .drag-handle:hover {
    background: var(--color-border);
  }
  .alias-form {
    padding: 16px;
    background: var(--color-bg-body);
    border: 1px dashed var(--color-border);
    border-radius: var(--radius-sm);
    margin-bottom: 16px;
  }
  .alias-row {
    padding: 10px 14px;
    background: var(--color-bg-body);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    transition: var(--transition);
  }
  .alias-row:hover {
    border-color: var(--color-border);
    box-shadow: var(--shadow-sm);
  }
  .btn-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: var(--radius-sm);
    border: none;
    background: transparent;
    cursor: pointer;
    transition: var(--transition);
  }
  .btn-icon:hover {
    background: var(--color-error-light);
  }
  @media (max-width: 768px) {
    .combo-card {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
