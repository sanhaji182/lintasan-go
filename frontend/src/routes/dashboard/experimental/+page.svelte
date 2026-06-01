<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import Spinner from '$lib/components/Spinner.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import CodexIcon from '$lib/components/icons/CodexIcon.svelte';
  import ClaudeIcon from '$lib/components/icons/ClaudeIcon.svelte';
  import GeminiIcon from '$lib/components/icons/GeminiIcon.svelte';
  import CopilotIcon from '$lib/components/icons/CopilotIcon.svelte';
  import { FlaskConical, CheckCircle, AlertTriangle, XCircle, ChevronDown, ChevronRight, Key, Lock, Unlock, Trash2, Wrench, Shield, Cpu, Terminal, ArrowRight, Zap } from 'lucide-svelte';

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

  interface CredentialStatus {
    provider: string;
    configured: boolean;
    source: string;
    masked_value: string;
    env_var: string;
    updated_at?: string;
  }

  let providers = $state<Provider[]>([]);
  let credentials = $state<Record<string, CredentialStatus>>({});
  let loading = $state(true);
  let error = $state('');
  let actionLoading = $state('');
  let expanded = $state<string | null>(null);
  let details = $state<Record<string, ProviderDetail>>({});
  let credentialInput = $state<Record<string, string>>({});
  let showCredForm = $state<string | null>(null);

  const summary = $derived({
    total: providers.length,
    active: providers.filter(p => p.state === 'active').length,
    admitted: providers.filter(p => p.state === 'admitted').length,
    proposed: providers.filter(p => p.state === 'proposed').length,
  });

  // Provider brand config
  const providerBrands: Record<string, {
    color: string; bg: string; gradient: string; border: string;
    company: string; tagline: string; icon: string;
    features: string[];
  }> = {
    codex: {
      color: '#10a37f', bg: 'rgba(16,163,127,0.06)',
      gradient: 'linear-gradient(135deg, #10a37f 0%, #0d8c6d 100%)',
      border: 'rgba(16,163,127,0.2)',
      company: 'OpenAI', tagline: 'Code generation & reasoning engine',
      icon: 'codex',
      features: ['Code gen', 'Reasoning', 'Agentic'],
    },
    'claude-code': {
      color: '#d97757', bg: 'rgba(217,119,87,0.06)',
      gradient: 'linear-gradient(135deg, #d97757 0%, #c4603f 100%)',
      border: 'rgba(217,119,87,0.2)',
      company: 'Anthropic', tagline: 'Conversational coding agent',
      icon: 'claude',
      features: ['Chat', 'Analysis', 'Safe'],
    },
    'gemini-cli': {
      color: '#4285f4', bg: 'rgba(66,133,244,0.06)',
      gradient: 'linear-gradient(135deg, #4285f4 0%, #3b7de9 100%)',
      border: 'rgba(66,133,244,0.2)',
      company: 'Google', tagline: 'Multimodal code assistant',
      icon: 'gemini',
      features: ['Multimodal', 'Fast', 'Context'],
    },
    copilot: {
      color: '#6366f1', bg: 'rgba(99,102,241,0.06)',
      gradient: 'linear-gradient(135deg, #6366f1 0%, #5558e6 100%)',
      border: 'rgba(99,102,241,0.2)',
      company: 'GitHub', tagline: 'AI pair programmer',
      icon: 'copilot',
      features: ['Inline', 'Chat', 'Workspace'],
    },
  };

  function getBrand(name: string) {
    return providerBrands[name] || {
      color: 'var(--color-primary)', bg: 'var(--color-primary-light)',
      gradient: 'linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary) 100%)',
      border: 'var(--color-border)',
      company: 'Unknown', tagline: 'Experimental provider',
      icon: 'default', features: [],
    };
  }

  function stateConfig(state: string) {
    switch (state) {
      case 'active': return { label: 'Active', color: '#22c55e', bg: 'rgba(34,197,94,0.12)', icon: 'check' };
      case 'admitted': return { label: 'Ready', color: '#3b82f6', bg: 'rgba(59,130,246,0.12)', icon: 'shield' };
      case 'deprecated': return { label: 'Deprecated', color: '#9ca3af', bg: 'rgba(156,163,175,0.12)', icon: 'x' };
      default: return { label: 'Setup', color: '#f59e0b', bg: 'rgba(245,158,11,0.12)', icon: 'alert' };
    }
  }

  onMount(async () => {
    await fetchProviders();
    await fetchCredentials();
  });

  async function fetchProviders() {
    loading = true; error = '';
    try {
      const res = await api.get<any>('/api/experimental/providers');
      providers = res.data || [];
    } catch (e: any) { error = e.message; }
    finally { loading = false; }
  }

  async function fetchCredentials() {
    try {
      const res = await api.get<any>('/api/experimental/credentials');
      const list: CredentialStatus[] = res.data || [];
      const map: Record<string, CredentialStatus> = {};
      for (const c of list) { map[c.provider] = c; }
      credentials = map;
    } catch (e: any) { /* non-fatal */ }
  }

  async function toggleDetail(name: string) {
    if (expanded === name) { expanded = null; return; }
    expanded = name;
    if (!details[name]) {
      try {
        const res = await api.get<any>(`/api/experimental/providers/${name}`);
        details[name] = res.data;
      } catch (e: any) { error = e.message; }
    }
  }

  async function admit(name: string) {
    actionLoading = name + ':admit';
    try {
      await api.post<any>(`/api/experimental/providers/${name}/admit`, {});
      await fetchProviders(); await fetchCredentials();
      if (expanded === name) { delete details[name]; expanded = null; }
    } catch (e: any) { error = e.message; }
    finally { actionLoading = ''; }
  }

  async function activate(name: string) {
    actionLoading = name + ':activate';
    try {
      await api.post<any>(`/api/experimental/providers/${name}/activate`, {});
      await fetchProviders();
      if (expanded === name) { delete details[name]; expanded = null; }
    } catch (e: any) { error = e.message; }
    finally { actionLoading = ''; }
  }

  async function deactivate(name: string) {
    actionLoading = name + ':deactivate';
    try {
      await api.post<any>(`/api/experimental/providers/${name}/deactivate`, {});
      await fetchProviders();
      if (expanded === name) { delete details[name]; expanded = null; }
    } catch (e: any) { error = e.message; }
    finally { actionLoading = ''; }
  }

  async function setCredential(name: string) {
    const value = credentialInput[name]?.trim();
    if (!value) return;
    actionLoading = name + ':cred';
    try {
      await api.put<any>(`/api/experimental/credentials/${name}`, { credential: value });
      credentialInput[name] = ''; showCredForm = null;
      await fetchCredentials(); await fetchProviders();
    } catch (e: any) { error = e.message; }
    finally { actionLoading = ''; }
  }

  async function deleteCredential(name: string) {
    actionLoading = name + ':cred-del';
    try {
      await api.delete<any>(`/api/experimental/credentials/${name}`);
      await fetchCredentials(); await fetchProviders();
    } catch (e: any) { error = e.message; }
    finally { actionLoading = ''; }
  }

  function fmtDate(d: string | null): string {
    if (!d) return '—';
    return new Date(d).toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
  }
</script>

<svelte:head><title>Experimental Providers — Lintasan</title></svelte:head>

<div style="animation: fadeInUp 0.4s ease-out;">
  <!-- Header -->
  <div class="flex items-center gap-3 mb-2">
    <div style="width: 40px; height: 40px; border-radius: 10px; background: linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%); display: flex; align-items: center; justify-content: center;">
      <FlaskConical size={20} color="white" />
    </div>
    <div>
      <h1 style="font-size: 22px; font-weight: 700; color: var(--color-fg-0); margin: 0;">Experimental Providers</h1>
      <p style="font-size: 12px; color: var(--color-fg-3); margin: 2px 0 0 0;">ACP-based coding agents — add and manage provider accounts</p>
    </div>
  </div>

  <!-- Stats Strip -->
  <div class="card mb-5" style="padding: 0; overflow: hidden;">
    <div class="grid grid-cols-4" style="gap: 1px; background: var(--color-border);">
      {#each [
        { label: 'PROVIDERS', value: summary.total, color: 'var(--color-fg-0)', icon: Cpu },
        { label: 'ACTIVE', value: summary.active, color: '#22c55e', icon: CheckCircle },
        { label: 'READY', value: summary.admitted, color: '#3b82f6', icon: Shield },
        { label: 'SETUP', value: summary.proposed, color: '#f59e0b', icon: AlertTriangle },
      ] as stat}
        <div class="text-center flex flex-col items-center justify-center" style="padding: 14px 16px; background: var(--color-bg-card); gap: 6px;">
          <svelte:component this={stat.icon} size={15} style="color: {stat.color}; opacity: 0.6;" />
          <div>
            <div style="font-size: 10px; font-weight: 500; color: var(--color-fg-3); text-transform: uppercase; letter-spacing: 0.8px; margin-bottom: 2px;">{stat.label}</div>
            <div style="font-size: 20px; font-weight: 700; color: {stat.color}; font-family: var(--font-mono);">{stat.value}</div>
          </div>
        </div>
      {/each}
    </div>
  </div>

  {#if error}
    <div style="background: rgba(239,68,68,0.1); color: #ef4444; padding: 10px 14px; border-radius: 8px; margin-bottom: 16px; font-size: 13px;">{error}</div>
  {/if}

  {#if loading}
    <div class="flex justify-center" style="padding: 60px 0;">
      <Spinner />
    </div>
  {:else if providers.length === 0}
    <EmptyState title="No Experimental Providers" description="Cohort-A providers will appear here once the framework is deployed." />
  {:else}
    <!-- Preset Card Grid -->
    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 20px;">
      {#each providers as p (p.name)}
        {@const brand = getBrand(p.name)}
        {@const sc = stateConfig(p.state)}
        {@const cred = credentials[p.name]}
        {@const isOpen = expanded === p.name}
        {@const det = details[p.name]}

        <div class="preset-card" style="--brand-color: {brand.color}; --brand-bg: {brand.bg}; --brand-border: {brand.border}; {isOpen ? 'box-shadow: 0 0 0 2px ' + brand.color + '30, 0 8px 24px rgba(0,0,0,0.12);' : ''}">
          <!-- Card Top Section -->
          <div class="card-top" style="background: {brand.bg};">
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center gap-3">
                <!-- Large Brand Icon -->
                <div class="brand-icon-lg" style="background: {brand.gradient};">
                  {#if brand.icon === 'codex'}
                    <CodexIcon size={32} />
                  {:else if brand.icon === 'claude'}
                    <ClaudeIcon size={32} />
                  {:else if brand.icon === 'gemini'}
                    <GeminiIcon size={32} />
                  {:else if brand.icon === 'copilot'}
                    <CopilotIcon size={32} />
                  {:else}
                    <Terminal size={24} color="white" />
                  {/if}
                </div>
                <div>
                  <div class="provider-name-lg">{p.name}</div>
                  <div class="company-badge" style="color: {brand.color}; background: {brand.color}15;">
                    {brand.company}
                  </div>
                </div>
              </div>
              <!-- State Badge -->
              <div class="state-badge" style="background: {sc.bg}; color: {sc.color};">
                {#if sc.icon === 'check'}<CheckCircle size={12} />
                {:else if sc.icon === 'shield'}<Shield size={12} />
                {:else if sc.icon === 'alert'}<AlertTriangle size={12} />
                {:else}<XCircle size={12} />{/if}
                {sc.label}
              </div>
            </div>
            <div class="tagline">{brand.tagline}</div>
            <!-- Feature Tags -->
            <div class="feature-tags">
              {#each brand.features as feat}
                <span class="feature-tag" style="color: {brand.color}; background: {brand.color}12; border: 1px solid {brand.color}20;">{feat}</span>
              {/each}
            </div>
          </div>

          <!-- Credential Section -->
          <div class="cred-section">
            <div class="cred-header">
              <Key size={13} style="color: var(--color-fg-3);" />
              <span style="font-size: 11px; color: var(--color-fg-3); font-weight: 500;">Credential</span>
            </div>
            <div class="cred-body">
              {#if cred?.configured}
                <div class="cred-status-row">
                  <div class="flex items-center gap-2">
                    <Lock size={13} style="color: #22c55e;" />
                    <span class="cred-source-badge" style="color: #22c55e; background: rgba(34,197,94,0.1);">{cred.source}</span>
                    {#if cred.masked_value}
                      <code class="cred-value">{cred.masked_value}</code>
                    {/if}
                  </div>
                  <div class="flex items-center gap-1.5">
                    <button class="cred-action" onclick={() => { showCredForm = p.name; }}>Update</button>
                    {#if cred.source === 'dashboard'}
                      <button class="cred-action danger" disabled={actionLoading === p.name + ':cred-del'} onclick={() => deleteCredential(p.name)}>
                        <Trash2 size={11} />
                      </button>
                    {/if}
                  </div>
                </div>
              {:else}
                <div class="cred-empty">
                  <Unlock size={14} style="color: var(--color-fg-3);" />
                  <span>No credential configured</span>
                </div>
              {/if}

              {#if showCredForm === p.name}
                <form class="cred-form" onsubmit={(e) => { e.preventDefault(); setCredential(p.name); }}>
                  <input
                    type="password"
                    class="cred-input"
                    placeholder={`Enter ${p.auth_env_var}`}
                    bind:value={credentialInput[p.name]}
                    autocomplete="off"
                  />
                  <div class="flex gap-2">
                    <button type="submit" class="cred-submit" style="background: {brand.gradient};" disabled={actionLoading === p.name + ':cred' || !credentialInput[p.name]?.trim()}>
                      {actionLoading === p.name + ':cred' ? 'Saving...' : 'Save'}
                    </button>
                    <button type="button" class="cred-cancel" onclick={() => { showCredForm = null; }}>Cancel</button>
                  </div>
                </form>
              {:else if !cred?.configured}
                <button class="cred-set-btn" style="color: {brand.color}; border-color: {brand.border};" onclick={() => { showCredForm = p.name; }}>
                  <Key size={13} />
                  Set {p.auth_env_var}
                </button>
              {/if}
            </div>
          </div>

          <!-- Info Row -->
          <div class="info-row">
            <div class="info-chip">
              <Wrench size={11} />
              <span>{p.capabilities?.length || 0} capabilities</span>
            </div>
            {#if p.validation_evidence}
              <div class="info-chip mono">{p.validation_evidence}</div>
            {/if}
          </div>

          <!-- Action Footer -->
          <div class="card-footer">
            {#if p.state === 'proposed'}
              <button
                class="primary-action"
                style="background: {brand.gradient};"
                disabled={actionLoading === p.name + ':admit' || !p.credential_set}
                onclick={() => admit(p.name)}
              >
                {#if actionLoading === p.name + ':admit'}
                  Admitting...
                {:else}
                  Admit Provider <ArrowRight size={14} />
                {/if}
              </button>
              {#if !p.credential_set}
                <div class="action-hint">
                  <AlertTriangle size={12} />
                  Set credential first
                </div>
              {/if}
            {:else if p.state === 'admitted'}
              <button class="primary-action activate" disabled={actionLoading === p.name + ':activate'} onclick={() => activate(p.name)}>
                {actionLoading === p.name + ':activate' ? 'Activating...' : 'Activate'} <Zap size={14} />
              </button>
              <button class="secondary-action" disabled={actionLoading === p.name + ':deactivate'} onclick={() => deactivate(p.name)}>
                Deactivate
              </button>
            {:else if p.state === 'active'}
              <div class="active-status">
                <CheckCircle size={16} style="color: #22c55e;" />
                <span>Live in routing</span>
              </div>
              <button class="secondary-action" disabled={actionLoading === p.name + ':deactivate'} onclick={() => deactivate(p.name)}>
                Deactivate
              </button>
            {/if}

            <button class="detail-toggle" onclick={() => toggleDetail(p.name)} title="Details">
              {#if isOpen}<ChevronDown size={16} />{:else}<ChevronRight size={16} />{/if}
            </button>
          </div>

          <!-- Expanded Detail -->
          {#if isOpen && det}
            <div class="detail-panel" style="border-top: 1px solid {brand.border};">
              <div class="detail-grid">
                <div class="detail-item">
                  <span class="detail-label">Executable</span>
                  <code class="detail-value">{det.default_path}{det.args?.length ? ' ' + det.args.join(' ') : ''}</code>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Auth Method</span>
                  <code class="detail-value">{det.auth_method_id || '—'}</code>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Admitted</span>
                  <span class="detail-value">{fmtDate(det.admitted_at)}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Activated</span>
                  <span class="detail-value">{fmtDate(det.activated_at)}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Isolation</span>
                  <span class="detail-value">Foreign vars: {det.foreign_auth_vars?.join(', ') || '—'}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Capabilities</span>
                  <div class="flex flex-wrap" style="gap: 4px;">
                    {#each (det.capabilities || []) as cap}
                      <span class="cap-chip" style="color: {brand.color}; background: {brand.bg}; border: 1px solid {brand.border};">{cap}</span>
                    {/each}
                    {#if !det.capabilities?.length}<span style="color: var(--color-fg-3); font-size: 12px;">—</span>{/if}
                  </div>
                </div>
              </div>

              {#if det.admission_report?.Results}
                <div class="gates-section">
                  <div class="gates-title">Admission Gates</div>
                  {#each det.admission_report.Results as gate}
                    <div class="gate-row">
                      {#if gate.Outcome === 'pass'}
                        <CheckCircle size={14} style="color: #22c55e; flex-shrink: 0;" />
                      {:else}
                        <XCircle size={14} style="color: #ef4444; flex-shrink: 0;" />
                      {/if}
                      <span class="gate-name">{gate.Gate}</span>
                      <span class="gate-reason">{gate.Reason}</span>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {:else if isOpen}
            <div class="detail-panel" style="padding: 16px; text-align: center;">
              <Spinner />
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .preset-card {
    background: var(--color-bg-card);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    overflow: hidden;
    transition: border-color 0.2s, box-shadow 0.2s, transform 0.15s;
  }
  .preset-card:hover {
    border-color: var(--brand-border, var(--color-border));
    box-shadow: 0 4px 20px rgba(0,0,0,0.08);
    transform: translateY(-2px);
  }

  .card-top {
    padding: 20px 20px 16px;
  }
  .brand-icon-lg {
    width: 52px; height: 52px; border-radius: 14px;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
  }
  .provider-name-lg {
    font-size: 18px; font-weight: 700; color: var(--color-fg-0);
    letter-spacing: -0.3px; margin-bottom: 4px;
  }
  .company-badge {
    display: inline-flex; align-items: center;
    padding: 2px 8px; border-radius: 6px;
    font-size: 10px; font-weight: 600; text-transform: uppercase;
    letter-spacing: 0.8px;
  }
  .tagline {
    font-size: 13px; color: var(--color-fg-2); margin-bottom: 10px;
    padding-left: 64px;
  }
  .feature-tags {
    display: flex; flex-wrap: wrap; gap: 6px;
    padding-left: 64px;
  }
  .feature-tag {
    padding: 3px 8px; border-radius: 6px;
    font-size: 10px; font-weight: 600;
    letter-spacing: 0.3px;
  }
  .state-badge {
    display: flex; align-items: center; gap: 5px;
    padding: 4px 10px; border-radius: 10px;
    font-size: 11px; font-weight: 700; text-transform: uppercase;
    letter-spacing: 0.5px; flex-shrink: 0;
  }

  .cred-section {
    padding: 14px 20px;
    border-top: 1px solid var(--color-border);
  }
  .cred-header {
    display: flex; align-items: center; gap: 6px;
    margin-bottom: 10px;
  }
  .cred-body {
    display: flex; flex-direction: column; gap: 10px;
  }
  .cred-status-row {
    display: flex; align-items: center; justify-content: space-between;
    flex-wrap: wrap; gap: 8px;
  }
  .cred-source-badge {
    padding: 2px 8px; border-radius: 6px;
    font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px;
  }
  .cred-value {
    font-size: 11px; color: var(--color-fg-2); background: var(--color-bg-3);
    padding: 3px 8px; border-radius: 4px; font-family: var(--font-mono);
  }
  .cred-empty {
    display: flex; align-items: center; gap: 8px;
    font-size: 12px; color: var(--color-fg-3); font-style: italic;
  }
  .cred-form {
    display: flex; flex-direction: column; gap: 8px;
  }
  .cred-input {
    width: 100%; padding: 8px 12px; border-radius: 8px;
    border: 1px solid var(--color-border);
    background: var(--color-bg-1); color: var(--color-fg-1);
    font-size: 13px; font-family: var(--font-mono);
    outline: none; box-sizing: border-box;
  }
  .cred-input:focus {
    border-color: var(--brand-color, var(--color-primary));
    box-shadow: 0 0 0 3px rgba(59,130,246,0.12);
  }
  .cred-submit {
    padding: 8px 16px; border-radius: 8px; border: none;
    color: white; font-size: 13px; font-weight: 600;
    cursor: pointer; transition: opacity 0.15s;
  }
  .cred-submit:disabled { opacity: 0.5; cursor: not-allowed; }
  .cred-cancel {
    padding: 8px 16px; border-radius: 8px; border: none;
    background: var(--color-bg-3); color: var(--color-fg-2);
    font-size: 13px; cursor: pointer;
  }
  .cred-action {
    padding: 4px 10px; border-radius: 6px; border: none;
    background: var(--color-bg-3); color: var(--color-fg-2);
    font-size: 11px; font-weight: 600; cursor: pointer;
    transition: background 0.15s;
  }
  .cred-action:hover { background: var(--color-border); }
  .cred-action.danger { color: #ef4444; }
  .cred-action.danger:hover { background: rgba(239,68,68,0.1); }
  .cred-action:disabled { opacity: 0.5; cursor: not-allowed; }
  .cred-set-btn {
    display: flex; align-items: center; gap: 6px;
    padding: 8px 14px; border-radius: 8px;
    border: 1px dashed; background: transparent;
    font-size: 12px; font-weight: 600; cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
    width: 100%; justify-content: center;
  }
  .cred-set-btn:hover { background: var(--brand-bg, var(--color-bg-3)); }

  .info-row {
    padding: 10px 20px;
    display: flex; flex-wrap: wrap; gap: 6px;
    border-top: 1px solid var(--color-border);
  }
  .info-chip {
    display: flex; align-items: center; gap: 4px;
    padding: 3px 8px; border-radius: 6px;
    background: var(--color-bg-3); color: var(--color-fg-2);
    font-size: 11px;
  }
  .info-chip.mono {
    font-family: var(--font-mono); font-size: 10px;
    color: var(--color-fg-3);
  }

  .card-footer {
    padding: 14px 20px;
    display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
    border-top: 1px solid var(--color-border);
  }
  .primary-action {
    flex: 1; padding: 10px 18px; border-radius: 10px; border: none;
    color: white; font-size: 13px; font-weight: 600;
    cursor: pointer; transition: opacity 0.15s, transform 0.1s;
    display: flex; align-items: center; justify-content: center; gap: 6px;
  }
  .primary-action:disabled { opacity: 0.5; cursor: not-allowed; }
  .primary-action:active:not(:disabled) { transform: scale(0.98); }
  .primary-action.activate { background: #22c55e; }
  .secondary-action {
    padding: 10px 16px; border-radius: 10px; border: none;
    background: var(--color-bg-3); color: var(--color-fg-2);
    font-size: 13px; font-weight: 600; cursor: pointer;
    transition: background 0.15s;
  }
  .secondary-action:hover { background: var(--color-border); }
  .secondary-action:disabled { opacity: 0.5; cursor: not-allowed; }
  .active-status {
    flex: 1; display: flex; align-items: center; gap: 6px;
    font-size: 13px; font-weight: 600; color: #22c55e;
  }
  .action-hint {
    display: flex; align-items: center; gap: 4px;
    font-size: 11px; color: #f59e0b;
  }
  .detail-toggle {
    width: 32px; height: 32px; border-radius: 8px;
    border: none; background: var(--color-bg-3);
    color: var(--color-fg-2); cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    transition: background 0.15s;
    margin-left: auto;
  }
  .detail-toggle:hover { background: var(--color-border); }

  .detail-panel {
    padding: 16px 20px;
    animation: fadeInScale 0.2s ease-out;
  }
  .detail-grid {
    display: grid; grid-template-columns: 1fr 1fr; gap: 12px;
    margin-bottom: 14px;
  }
  .detail-item { display: flex; flex-direction: column; gap: 3px; }
  .detail-label {
    font-size: 10px; color: var(--color-fg-3);
    text-transform: uppercase; letter-spacing: 0.8px;
  }
  .detail-value { font-size: 12px; color: var(--color-fg-1); }
  code.detail-value {
    background: var(--color-bg-3); padding: 3px 8px;
    border-radius: 4px; font-family: var(--font-mono);
  }
  .cap-chip {
    padding: 2px 7px; border-radius: 4px;
    font-size: 10px; font-weight: 600;
  }
  .gates-section { margin-top: 12px; }
  .gates-title {
    font-size: 11px; font-weight: 600; color: var(--color-fg-2);
    margin-bottom: 8px; text-transform: uppercase; letter-spacing: 0.5px;
  }
  .gate-row {
    display: flex; align-items: flex-start; gap: 6px;
    font-size: 12px; padding: 4px 0;
  }
  .gate-name { font-weight: 600; color: var(--color-fg-1); min-width: 70px; }
  .gate-reason { color: var(--color-fg-3); word-break: break-word; }
</style>
