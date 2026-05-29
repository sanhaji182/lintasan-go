<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import {
    ArrowRight,
    CheckCircle2,
    Eye,
    EyeOff,
    Loader2,
    Lock,
    Shield,
    Sparkles,
    User
  } from 'lucide-svelte';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);
  let checkingSession = $state(true);
  let showPassword = $state(false);
  let mounted = $state(false);

  onMount(async () => {
    mounted = true;
    const token = localStorage.getItem('lintasan_token');

    if (!token) {
      checkingSession = false;
      return;
    }

    try {
      await api.get('/api/auth/me');
      await goto('/dashboard');
    } catch {
      localStorage.removeItem('lintasan_token');
      localStorage.removeItem('lintasan_user');
      checkingSession = false;
    }
  });

  async function handleLogin() {
    if (!username.trim() || !password.trim()) {
      error = 'Username dan password wajib diisi';
      return;
    }

    loading = true;
    error = '';

    try {
      const data = await api.post<{ token: string; user: { id: string; username: string; role: string } }>(
        '/api/auth/login',
        { username: username.trim(), password: password.trim() }
      );

      if (data.token) {
        localStorage.setItem('lintasan_token', data.token);
        localStorage.setItem('lintasan_user', JSON.stringify(data.user));
      }

      await goto('/dashboard');
    } catch (e: any) {
      error = e.message || 'Login gagal. Periksa kredensial Anda.';
      password = '';
    } finally {
      loading = false;
    }
  }

  function onSubmit(event: SubmitEvent) {
    event.preventDefault();
    if (!loading) handleLogin();
  }

  const formDisabled = $derived(loading || checkingSession);
</script>

<svelte:head>
  <title>Login — Lintasan</title>
  <meta name="description" content="Masuk ke dashboard Lintasan untuk kelola AI gateway, routing, observability, dan keamanan." />
</svelte:head>

<div class="login-shell" class:mounted>
  <div class="bg-grid"></div>
  <div class="blob blob-a"></div>
  <div class="blob blob-b"></div>

  <div class="login-layout">
    <section class="login-info card-soft">
      <div class="brand">
        <div class="brand-icon" aria-hidden="true">
          <Shield size={24} stroke-width={1.7} />
        </div>
        <div>
          <p class="kicker">Lintasan Dashboard</p>
          <h1>Secure gateway control untuk tim AI engineering.</h1>
        </div>
      </div>

      <p class="lead">
        Akses monitoring, routing policy, dan analytics dalam satu workspace dengan tema dark-ocean modern.
      </p>

      <div class="trust-list">
        <div class="trust-item">
          <CheckCircle2 size={16} />
          <span>JWT authentication + scoped API keys</span>
        </div>
        <div class="trust-item">
          <CheckCircle2 size={16} />
          <span>Realtime logs dan observability</span>
        </div>
        <div class="trust-item">
          <CheckCircle2 size={16} />
          <span>Fallback routing siap production</span>
        </div>
      </div>
    </section>

    <section class="login-card card-soft" aria-busy={formDisabled}>
      <div class="card-head">
        <div class="head-label">
          <Sparkles size={14} />
          Welcome back
        </div>
        <h2>Sign in to continue</h2>
        <p>Masuk untuk melanjutkan ke dashboard Lintasan.</p>
      </div>

      {#if checkingSession}
        <div class="session-check" role="status" aria-live="polite">
          <span class="spin"><Loader2 size={16} /></span>
          <span>Memeriksa sesi aktif...</span>
        </div>
      {/if}

      <form class="form" onsubmit={onSubmit} novalidate>
        <div class="field">
          <label for="username">Username</label>
          <div class="input-wrap">
            <span class="icon"><User size={15} /></span>
            <input
              id="username"
              class="text-input"
              type="text"
              autocomplete="username"
              placeholder="Masukkan username"
              bind:value={username}
              disabled={formDisabled}
              required
            />
          </div>
        </div>

        <div class="field">
          <label for="password">Password</label>
          <div class="input-wrap">
            <span class="icon"><Lock size={15} /></span>
            <input
              id="password"
              class="text-input"
              type={showPassword ? 'text' : 'password'}
              autocomplete="current-password"
              placeholder="Masukkan password"
              bind:value={password}
              disabled={formDisabled}
              required
              minlength="1"
            />
            <button
              type="button"
              class="toggle"
              onclick={() => (showPassword = !showPassword)}
              disabled={formDisabled}
              aria-label={showPassword ? 'Sembunyikan password' : 'Tampilkan password'}
            >
              {#if showPassword}
                <EyeOff size={16} />
              {:else}
                <Eye size={16} />
              {/if}
            </button>
          </div>
        </div>

        {#if error}
          <div class="error" role="alert" aria-live="assertive">
            {error}
          </div>
        {/if}

        <button
          class="submit"
          type="submit"
          disabled={formDisabled || !username.trim() || !password.trim()}
        >
          {#if loading}
            <span class="spin"><Loader2 size={16} /></span>
            <span>Signing in...</span>
          {:else}
            <span>Sign In</span>
            <ArrowRight size={16} />
          {/if}
        </button>
      </form>

      <p class="footer-note">
        <Shield size={12} />
        Session akan divalidasi otomatis sebelum akses dashboard.
      </p>
    </section>
  </div>
</div>

<style>
  .login-shell {
    position: fixed;
    inset: 0;
    overflow: auto;
    background: linear-gradient(155deg, #070c17 0%, #0c1424 44%, #0f172a 100%);
    color: #e2e8f0;
    opacity: 0;
    transition: opacity 0.4s ease;
  }

  .login-shell.mounted { opacity: 1; }

  .bg-grid {
    position: fixed;
    inset: 0;
    background-image:
      linear-gradient(rgba(148,163,184,0.06) 1px, transparent 1px),
      linear-gradient(90deg, rgba(148,163,184,0.06) 1px, transparent 1px);
    background-size: 42px 42px;
    pointer-events: none;
  }

  .blob {
    position: fixed;
    border-radius: 999px;
    filter: blur(84px);
    pointer-events: none;
  }

  .blob-a {
    width: 380px;
    height: 380px;
    top: -120px;
    right: -100px;
    background: rgba(59,130,246,0.33);
    animation: floatY 8s ease-in-out infinite;
  }

  .blob-b {
    width: 300px;
    height: 300px;
    bottom: -120px;
    left: -80px;
    background: rgba(99,102,241,0.31);
    animation: floatY 10s ease-in-out infinite reverse;
  }

  .login-layout {
    position: relative;
    z-index: 2;
    width: min(1050px, calc(100% - 38px));
    margin: min(7vh, 64px) auto;
    display: grid;
    grid-template-columns: 1fr 420px;
    gap: 16px;
    align-items: stretch;
  }

  .card-soft {
    border-radius: 18px;
    border: 1px solid rgba(148,163,184,0.2);
    background: rgba(15,23,42,0.72);
    box-shadow: 0 20px 42px rgba(2,6,23,0.44);
    backdrop-filter: blur(16px);
  }

  .login-info {
    padding: 30px;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 26px;
  }

  .brand {
    display: flex;
    gap: 14px;
    align-items: flex-start;
  }

  .brand-icon {
    width: 45px;
    height: 45px;
    border-radius: 12px;
    display: grid;
    place-items: center;
    color: #ffffff;
    background: linear-gradient(135deg, #3c50e0 0%, #6366f1 100%);
    box-shadow: 0 10px 24px rgba(60,80,224,0.4);
  }

  .kicker {
    margin: 0;
    text-transform: uppercase;
    font-size: 11px;
    letter-spacing: 0.08em;
    color: #93c5fd;
    font-weight: 700;
  }

  h1 {
    margin: 9px 0 0;
    font-size: clamp(30px, 4vw, 40px);
    line-height: 1.14;
    letter-spacing: -0.03em;
    color: #f8fafc;
  }

  .lead {
    margin: 0;
    font-size: 15px;
    line-height: 1.75;
    color: #cbd5e1;
    max-width: 52ch;
  }

  .trust-list {
    display: grid;
    gap: 10px;
  }

  .trust-item {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 11px 12px;
    border-radius: 11px;
    border: 1px solid rgba(96,165,250,0.3);
    background: rgba(30,64,175,0.17);
    color: #bfdbfe;
    font-size: 13px;
    font-weight: 500;
  }

  .login-card {
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 15px;
  }

  .head-label {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    border-radius: 999px;
    padding: 6px 10px;
    border: 1px solid rgba(96,165,250,0.25);
    background: rgba(30,64,175,0.16);
    color: #bfdbfe;
    font-size: 11px;
    font-weight: 600;
    width: fit-content;
  }

  h2 {
    margin: 12px 0 0;
    font-size: 25px;
    line-height: 1.2;
    letter-spacing: -0.03em;
    color: #f8fafc;
  }

  .card-head p {
    margin: 6px 0 0;
    color: #94a3b8;
    font-size: 13px;
  }

  .session-check {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: #93c5fd;
    border: 1px dashed rgba(96,165,250,0.4);
    background: rgba(30,64,175,0.12);
    border-radius: 10px;
    padding: 9px 10px;
  }

  .form {
    display: flex;
    flex-direction: column;
    gap: 13px;
  }

  .field label {
    display: block;
    margin-bottom: 6px;
    color: #cbd5e1;
    font-size: 12px;
    font-weight: 600;
  }

  .input-wrap {
    position: relative;
    display: flex;
    align-items: center;
  }

  .icon {
    position: absolute;
    left: 12px;
    color: #64748b;
    pointer-events: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  .icon :global(svg) {
    width: 15px;
    height: 15px;
  }

  .text-input {
    width: 100%;
    border: 1px solid rgba(148,163,184,0.28);
    background: rgba(15,23,42,0.9);
    color: #f1f5f9;
    border-radius: 10px;
    min-height: 43px;
    padding: 0 40px 0 36px;
    font-size: 14px;
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
  }

  .text-input::placeholder { color: #64748b; }

  .text-input:focus {
    outline: none;
    border-color: #4f63e8;
    box-shadow: 0 0 0 3px rgba(79,99,232,0.24);
  }

  .text-input:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }

  .toggle {
    position: absolute;
    right: 8px;
    border: none;
    border-radius: 7px;
    width: 29px;
    height: 29px;
    display: grid;
    place-items: center;
    cursor: pointer;
    color: #94a3b8;
    background: transparent;
    transition: background 0.15s ease, color 0.15s ease;
  }

  .toggle:hover:not(:disabled) {
    background: rgba(148,163,184,0.14);
    color: #e2e8f0;
  }

  .toggle:disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }

  .error {
    border-radius: 10px;
    border: 1px solid rgba(248,113,113,0.45);
    background: rgba(127,29,29,0.25);
    color: #fecaca;
    font-size: 13px;
    padding: 9px 11px;
    line-height: 1.5;
  }

  .submit {
    margin-top: 2px;
    width: 100%;
    min-height: 43px;
    border: none;
    border-radius: 10px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    font-size: 14px;
    font-weight: 600;
    color: #fff;
    background: linear-gradient(135deg, #3c50e0 0%, #5568eb 100%);
    cursor: pointer;
    transition: transform 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease;
    box-shadow: 0 12px 22px rgba(60,80,224,0.3);
  }

  .submit:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 15px 28px rgba(60,80,224,0.36);
  }

  .submit:disabled {
    opacity: 0.58;
    cursor: not-allowed;
    transform: none;
    box-shadow: none;
  }

  .footer-note {
    margin: 2px 0 0;
    font-size: 12px;
    color: #94a3b8;
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }

  .spin { animation: spin 0.9s linear infinite; }

  @media (max-width: 980px) {
    .login-layout {
      grid-template-columns: 1fr;
      width: min(680px, calc(100% - 26px));
    }

    .login-info {
      padding: 22px;
      gap: 18px;
    }

    h1 { font-size: 30px; }
  }

  @media (max-width: 620px) {
    .login-layout {
      margin: 18px auto;
      width: calc(100% - 20px);
      gap: 12px;
    }

    .login-info {
      padding: 18px;
    }

    .login-card {
      padding: 18px;
    }

    h2 {
      font-size: 22px;
    }
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  @keyframes floatY {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-14px); }
  }
</style>
