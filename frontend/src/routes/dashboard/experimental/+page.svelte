<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import Spinner from '$lib/components/Spinner.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import { FlaskConical, Shield, CheckCircle, AlertTriangle, XCircle, ChevronDown, ChevronRight, Key, Cpu, Wrench } from 'lucide-svelte';

  interface Provider {
    name: string;
    track: string;
    state: string;
    risk_badge: string;
    auth_env_var: string;
    credential_set: boolean;
    validation_evidence: string;
    admitted_at: string | null;
    activated_at: string | null;
    capabilities: string[];
    descriptor: any;
  }

  interface GateResult {
    Gate: string;
    Outcome: string;
    Reason: string;
  }

  interface ProviderDetail extends Provider {
    auth_method_id: string;
    deactivated_at: string | null;
    default_path: string;
    args: string[] | null;
    foreign_auth_vars: string[];
    admission_report: { Go?: boolean; Results?: GateResult[] } | null;
  }

  let providers = $state<Provider[]>([]);
  let loading = $state(true);
  let error = $state('');
  let actionLoading = $state('');
  let expanded = $state<string | null>(null);
  let detail = $state<ProviderDetail | null>(null);

  const summary = $derived({
    total: providers.length,
    active: providers.filter(p => p.state === 'active').length,
    admitted: providers.filter(p => p.state === 'admitted').length,
    proposed: providers.filter(p => p.state === 'proposed').length,
    deprecated: providers.filter(p => p.state === 'deprecated').length,
  });

  onMount(fetchProviders);

  async function fetchProviders() {
    loading = true;
    error = '';
    try {
      const res = await api.get<any>('/api/experimental/providers');
      providers = res.data || [];
    } catch (e: any) { error = e.message; }
    finally { loading = false; }
  }

  async function loadDetail(name: string) {
    if (expanded === name) { expanded = null; detail = null; return; }
    expanded = name;
    detail = null;
    try {
      const res = await api.get<any>(`/api/experimental/providers/${name}`);
      detail = res.data || null;
    } catch (e: any) { error = e.message; }
  }

  async function admit(name: string) {
    actionLoading = name + ':admit';
    try {
      await api.post<any>(`/api/experimental/providers/${name}/admit`, {});
      await fetchProviders();
      if (expanded === name) await loadDetail(name);
    } catch (e: any) { error = e.message; }
    finally { actionLoading = ''; }
  }

  async function activate(name: string) {
    actionLoading = name + ':activate';
    try {
      await api.post<any>(`/api/experimental/providers/${name}/activate`, {});
      await fetchProviders();
      if (expanded === name) await loadDetail(name);
    } catch (e: any) { error = e.message; }
    finally { actionLoading = ''; }
  }

  async function deactivate(name: string) {
    actionLoading = name + ':deactivate';
    try {
      await api.post<any>(`/api/experimental/providers/${name}/deactivate`, {});
      await fetchProviders();
      if (expanded === name) await loadDetail(name);
    } catch (e: any) { error = e.message; }
    finally { actionLoading = ''; }
  }

  function stateColor(state: string): string {
    switch (state) {
      case 'active': return 'var(--color-success, #22c55e)';
      case 'admitted': return 'var(--color-primary, #3b82f6)';
      case 'deprecated': return 'var(--color-fg-3, #9ca3af)';
      default: return 'var(--color-warning, #f59e0b)';
    }
  }

  function badgeStyle(state: string): string {
    const bg = state === 'active' ? 'rgba(34,197,94,0.1)' :
               state === 'admitted' ? 'rgba(59,130,246,0.1)' :
               state === 'deprecated' ? 'rgba(156,163,175,0.1)' :
               'rgba(245,158,11,0.1)';
    return `background:${bg}; color:${stateColor(state)}; font-weight:600; padding:3px 10px; border-radius:12px; font-size:11px; text-transform:uppercase; letter-spacing:0.5px;`;
  }

  function fmtDate(d: string | null): string {
    if (!d) return '—';
    return new Date(d).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' });
  }
</script>

<svelte:head><title>Experimental Providers — Lintasan</title></svelte:head>

<div class="page-container">
  <div class="page-header">
    <div class="flex items-center gap-3">
      <FlaskConical size={24} style="color: var(--color-primary)" />
      <div>
        <h1 class="page-title">Experimental Providers</h1>
        <p class="page-subtitle">ACP-based coding agents managed through the Generic Provider Framework</p>
      </div>
    </div>
  </div>

  <!-- Summary Strip -->
  <div class="summary-strip">
    <div class="stat-card">
      <div class="stat-value">{summary.total}</div>
      <div class="stat-label">Total</div>
    </div>
    <div class="stat-card">
      <div class="stat-value" style="color: var(--color-success)">{summary.active}</div>
      <div class="stat-label">Active</div>
    </div>
    <div class="stat-card">
      <div class="stat-value" style="color: var(--color-primary)">{summary.admitted}</div>
      <div class="stat-label">Admitted</div>
    </div>
    <div class="stat-card">
      <div class="stat-value" style="color: var(--color-warning)">{summary.proposed}</div>
      <div class="stat-label">Pending</div>
    </div>
    <div class="stat-card">
      <div class="stat-value" style="color: var(--color-fg-3)">{summary.deprecated}</div>
      <div class="stat-label">Deprecated</div>
    </div>
  </div>

  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  {#if loading}
    <Spinner />
  {:else if providers.length === 0}
    <EmptyState title="No Experimental Providers" description="Cohort-A providers will appear here once the framework is deployed." />
  {:else}
    <div class="provider-list">
      {#each providers as p}
        <div class="provider-card">
          <button class="provider-header" onclick={() => loadDetail(p.name)}>
            <div class="provider-left">
              {#if expanded === p.name}
                <ChevronDown size={16} />
              {:else}
                <ChevronRight size={16} />
              {/if}
              <Cpu size={18} style="color: var(--color-fg-2)" />
              <span class="provider-name">{p.name}</span>
              <span style={badgeStyle(p.state)}>{p.state}</span>
            </div>
            <div class="provider-right">
              <span class="cred-indicator" title={p.auth_env_var}>
                <Key size={14} />
                {#if p.credential_set}
                  <CheckCircle size={14} style="color: var(--color-success)" />
                {:else}
                  <AlertTriangle size={14} style="color: var(--color-warning)" />
                {/if}
                <span class="cred-label">{p.auth_env_var}</span>
              </span>
              {#if p.capabilities?.length}
                <span class="caps-count" title={p.capabilities.join(', ')}>
                  <Wrench size={14} /> {p.capabilities.length}
                </span>
              {/if}
            </div>
          </button>

          <!-- Actions row -->
          <div class="provider-actions">
            {#if p.state === 'proposed'}
              <button class="btn-sm btn-primary" disabled={actionLoading === p.name + ':admit' || !p.credential_set} onclick={() => admit(p.name)}>
                {actionLoading === p.name + ':admit' ? 'Admitting...' : 'Admit'}
              </button>
              {#if !p.credential_set}
                <span class="hint">Set {p.auth_env_var} to enable admission</span>
              {/if}
            {:else if p.state === 'admitted'}
              <button class="btn-sm btn-success" disabled={actionLoading === p.name + ':activate'} onclick={() => activate(p.name)}>
                {actionLoading === p.name + ':activate' ? 'Activating...' : 'Activate'}
              </button>
              <button class="btn-sm btn-muted" disabled={actionLoading === p.name + ':deactivate'} onclick={() => deactivate(p.name)}>
                Deactivate
              </button>
            {:else if p.state === 'active'}
              <button class="btn-sm btn-muted" disabled={actionLoading === p.name + ':deactivate'} onclick={() => deactivate(p.name)}>
                Deactivate
              </button>
            {/if}
            {#if p.validation_evidence}
              <span class="evidence-tag">{p.validation_evidence}</span>
            {/if}
          </div>

          <!-- Expanded detail -->
          {#if expanded === p.name && detail}
            <div class="provider-detail">
              <div class="detail-grid">
                <div class="detail-item">
                  <span class="detail-label">Executable</span>
                  <code>{detail.default_path}{detail.args?.length ? ' ' + detail.args.join(' ') : ''}</code>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Auth Method</span>
                  <code>{detail.auth_method_id || '—'}</code>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Admitted</span>
                  <span>{fmtDate(detail.admitted_at)}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Activated</span>
                  <span>{fmtDate(detail.activated_at)}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Isolation</span>
                  <span>Foreign vars: {detail.foreign_auth_vars?.join(', ') || '—'}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Capabilities</span>
                  <span>{detail.capabilities?.join(', ') || '—'}</span>
                </div>
              </div>

              <!-- Admission Report (gate results) -->
              {#if detail.admission_report?.Results}
                <div class="gate-results">
                  <h4>Admission Gates</h4>
                  <div class="gate-grid">
                    {#each detail.admission_report.Results as gate}
                      <div class="gate-item">
                        <span class="gate-icon">
                          {#if gate.Outcome === 'pass'}
                            <CheckCircle size={16} style="color: var(--color-success)" />
                          {:else}
                            <XCircle size={16} style="color: var(--color-error, #ef4444)" />
                          {/if}
                        </span>
                        <span class="gate-name">{gate.Gate}</span>
                        <span class="gate-reason">{gate.Reason}</span>
                      </div>
                    {/each}
                  </div>
                </div>
              {/if}
            </div>
          {:else if expanded === p.name}
            <div class="provider-detail"><Spinner /></div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .page-container { padding: 24px; max-width: 1200px; margin: 0 auto; }
  .page-header { margin-bottom: 24px; }
  .page-title { font-size: 22px; font-weight: 700; color: var(--color-fg-1); margin: 0; }
  .page-subtitle { font-size: 13px; color: var(--color-fg-3); margin-top: 4px; }
  .flex { display: flex; }
  .items-center { align-items: center; }
  .gap-3 { gap: 12px; }

  .summary-strip {
    display: flex; gap: 12px; margin-bottom: 24px; flex-wrap: wrap;
  }
  .stat-card {
    background: var(--color-bg-2); border: 1px solid var(--color-border-light);
    border-radius: 10px; padding: 14px 20px; min-width: 100px; text-align: center;
  }
  .stat-value { font-size: 24px; font-weight: 700; color: var(--color-fg-1); }
  .stat-label { font-size: 11px; color: var(--color-fg-3); text-transform: uppercase; letter-spacing: 0.5px; margin-top: 2px; }

  .error-banner {
    background: rgba(239,68,68,0.1); color: var(--color-error, #ef4444);
    padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; font-size: 13px;
  }

  .provider-list { display: flex; flex-direction: column; gap: 12px; }
  .provider-card {
    background: var(--color-bg-2); border: 1px solid var(--color-border-light);
    border-radius: 12px; overflow: hidden;
  }
  .provider-header {
    display: flex; align-items: center; justify-content: space-between;
    width: 100%; padding: 16px 20px; border: none; background: none;
    cursor: pointer; text-align: left; gap: 12px;
  }
  .provider-header:hover { background: var(--color-bg-3); }
  .provider-left { display: flex; align-items: center; gap: 10px; }
  .provider-right { display: flex; align-items: center; gap: 16px; }
  .provider-name { font-weight: 600; font-size: 15px; color: var(--color-fg-1); }

  .cred-indicator { display: flex; align-items: center; gap: 4px; font-size: 12px; color: var(--color-fg-3); }
  .cred-label { font-family: monospace; font-size: 11px; }
  .caps-count { display: flex; align-items: center; gap: 4px; font-size: 12px; color: var(--color-fg-3); }

  .provider-actions {
    display: flex; align-items: center; gap: 10px; padding: 0 20px 14px;
  }
  .btn-sm {
    padding: 6px 14px; border-radius: 6px; font-size: 12px; font-weight: 600;
    border: none; cursor: pointer; transition: opacity 0.15s;
  }
  .btn-sm:disabled { opacity: 0.5; cursor: not-allowed; }
  .btn-primary { background: var(--color-primary); color: white; }
  .btn-success { background: var(--color-success, #22c55e); color: white; }
  .btn-muted { background: var(--color-bg-3); color: var(--color-fg-2); }
  .hint { font-size: 11px; color: var(--color-warning); }
  .evidence-tag {
    font-size: 11px; color: var(--color-fg-3); background: var(--color-bg-3);
    padding: 3px 8px; border-radius: 4px; font-family: monospace;
  }

  .provider-detail { padding: 0 20px 20px; }
  .detail-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 12px; margin-bottom: 16px; }
  .detail-item { display: flex; flex-direction: column; gap: 2px; }
  .detail-label { font-size: 11px; color: var(--color-fg-3); text-transform: uppercase; letter-spacing: 0.5px; }
  .detail-item code { font-size: 13px; color: var(--color-fg-1); background: var(--color-bg-3); padding: 2px 6px; border-radius: 4px; }
  .detail-item span { font-size: 13px; color: var(--color-fg-2); }

  .gate-results h4 { font-size: 13px; font-weight: 600; color: var(--color-fg-2); margin: 0 0 10px; }
  .gate-grid { display: flex; flex-direction: column; gap: 8px; }
  .gate-item { display: flex; align-items: flex-start; gap: 8px; font-size: 12px; }
  .gate-icon { flex-shrink: 0; margin-top: 1px; }
  .gate-name { font-weight: 600; color: var(--color-fg-1); min-width: 90px; }
  .gate-reason { color: var(--color-fg-3); word-break: break-word; }
</style>
