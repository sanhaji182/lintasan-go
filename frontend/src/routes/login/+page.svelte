<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import {
    Lock, Eye, EyeOff, ArrowRight, Shield, Loader
  } from 'lucide-svelte';

  let password = $state('');
  let error = $state('');
  let loading = $state(false);
  let showPassword = $state(false);
  let mounted = $state(false);

  onMount(() => {
    mounted = true;
    // Check if already authenticated
    const token = localStorage.getItem('auth_token');
    if (token) {
      goto('/dashboard');
    }
  });

  async function handleLogin() {
    if (!password.trim()) {
      error = 'Please enter your password';
      return;
    }

    loading = true;
    error = '';

    try {
      const data = await api.post<{ token: string }>('/api/auth/login', {
        password: password.trim()
      });

      if (data.token) {
        localStorage.setItem('auth_token', data.token);
      }

      await goto('/dashboard');
    } catch (e: any) {
      error = e.message || 'Invalid password. Please try again.';
      password = '';
    }

    loading = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleLogin();
    }
  }
</script>

<svelte:head>
  <title>Login — Lintasan</title>
</svelte:head>

<div class="login-page" class:mounted>
  <!-- Decorative background elements -->
  <div class="bg-grid"></div>
  <div class="bg-glow bg-glow-1"></div>
  <div class="bg-glow bg-glow-2"></div>

  <div class="login-container">
    <!-- Logo / Brand -->
    <div class="brand">
      <div class="logo-icon">
        <Shield size={28} stroke-width={1.5} />
      </div>
      <h1 class="brand-name">Lintasan</h1>
      <p class="brand-tagline">AI Gateway Management</p>
    </div>

    <!-- Login Card -->
    <div class="login-card">
      <div class="card-header">
        <div class="flex items-center gap-2">
          <Lock size={16} style="color: var(--color-primary);" />
          <span style="font-size: 15px; font-weight: 600; color: var(--color-fg-0);">Sign In</span>
        </div>
        <p style="font-size: 13px; color: var(--color-fg-3); margin-top: 4px;">
          Enter your admin password to continue
        </p>
      </div>

      <div class="card-body">
        <!-- Password Input -->
        <div class="input-wrapper">
          <!-- svelte-ignore a11y_label_has_associated_control -->
          <label style="font-size: 12px; font-weight: 500; color: var(--color-fg-2); display: block; margin-bottom: 6px;">
            Password
          </label>
          <div class="password-field">
            <Lock size={14} class="password-icon" />
            <input
              class="password-input"
              type={showPassword ? 'text' : 'password'}
              placeholder="Enter your password"
              bind:value={password}
              onkeydown={handleKeydown}
              disabled={loading}
              autofocus
            />
            <button
              class="eye-toggle"
              onclick={() => showPassword = !showPassword}
              tabindex="-1"
            >
              {#if showPassword}
                <EyeOff size={16} style="color: var(--color-fg-3);" />
              {:else}
                <Eye size={16} style="color: var(--color-fg-3);" />
              {/if}
            </button>
          </div>
        </div>

        <!-- Error Message -->
        {#if error}
          <div class="error-message" style="animation: fadeInUp 0.2s ease-out;">
            {error}
          </div>
        {/if}

        <!-- Submit Button -->
        <button
          class="login-btn"
          onclick={handleLogin}
          disabled={loading || !password.trim()}
        >
          {#if loading}
            <Loader size={16} class="spin-icon" />
            <span>Authenticating...</span>
          {:else}
            <span>Sign In</span>
            <ArrowRight size={16} />
          {/if}
        </button>
      </div>

      <!-- Footer -->
      <div class="card-footer">
        <div class="flex items-center gap-1.5" style="font-size: 11px; color: var(--color-fg-3);">
          <Shield size={12} />
          Secured with encrypted authentication
        </div>
      </div>
    </div>

    <!-- Version -->
    <div class="version-text">
      Lintasan Gateway v1.0
    </div>
  </div>
</div>

<style>
  .login-page {
    position: fixed;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #0a0e1a 0%, #111827 40%, #0f172a 100%);
    overflow: hidden;
    opacity: 0;
    transition: opacity 0.5s ease-out;
  }
  .login-page.mounted {
    opacity: 1;
  }

  .bg-grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(rgba(60, 80, 224, 0.03) 1px, transparent 1px),
      linear-gradient(90deg, rgba(60, 80, 224, 0.03) 1px, transparent 1px);
    background-size: 40px 40px;
    pointer-events: none;
  }

  .bg-glow {
    position: absolute;
    border-radius: 50%;
    filter: blur(80px);
    pointer-events: none;
  }
  .bg-glow-1 {
    width: 500px;
    height: 500px;
    background: rgba(60, 80, 224, 0.08);
    top: -200px;
    right: -100px;
    animation: float 8s ease-in-out infinite;
  }
  .bg-glow-2 {
    width: 400px;
    height: 400px;
    background: rgba(139, 92, 246, 0.06);
    bottom: -150px;
    left: -100px;
    animation: float 10s ease-in-out infinite reverse;
  }

  .login-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 100%;
    max-width: 400px;
    padding: 20px;
    z-index: 1;
  }

  .brand {
    text-align: center;
    margin-bottom: 32px;
    animation: fadeInUp 0.5s ease-out;
  }

  .logo-icon {
    width: 56px;
    height: 56px;
    border-radius: 16px;
    background: linear-gradient(135deg, #3c50e0, #8b5cf6);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    margin: 0 auto 16px;
    box-shadow: 0 8px 32px rgba(60, 80, 224, 0.3);
  }

  .brand-name {
    font-size: 28px;
    font-weight: 700;
    color: #f1f5f9;
    letter-spacing: -0.5px;
    margin: 0;
  }

  .brand-tagline {
    font-size: 13px;
    color: #64748b;
    margin-top: 4px;
  }

  .login-card {
    width: 100%;
    background: rgba(26, 35, 50, 0.8);
    backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 16px;
    overflow: hidden;
    animation: fadeInUp 0.6s ease-out 0.1s both;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.4);
  }

  .card-header {
    padding: 24px 24px 0;
  }

  .card-body {
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .card-footer {
    padding: 16px 24px;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    text-align: center;
  }

  .password-field {
    position: relative;
    display: flex;
    align-items: center;
  }

  .password-field :global(.password-icon) {
    position: absolute;
    left: 12px;
    color: #64748b;
    pointer-events: none;
  }

  .password-input {
    width: 100%;
    padding: 10px 40px 10px 36px;
    background: rgba(15, 23, 42, 0.6);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 8px;
    font-size: 14px;
    color: #f1f5f9;
    transition: all 0.2s ease;
    outline: none;
  }
  .password-input::placeholder {
    color: #475569;
  }
  .password-input:focus {
    border-color: #3c50e0;
    box-shadow: 0 0 0 3px rgba(60, 80, 224, 0.2);
  }
  .password-input:disabled {
    opacity: 0.5;
  }

  .eye-toggle {
    position: absolute;
    right: 8px;
    background: none;
    border: none;
    cursor: pointer;
    padding: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    transition: all 0.2s ease;
  }
  .eye-toggle:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  .error-message {
    padding: 10px 14px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 8px;
    font-size: 13px;
    color: #ef4444;
    font-weight: 500;
  }

  .login-btn {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 11px 20px;
    background: linear-gradient(135deg, #3c50e0, #4f63e8);
    color: white;
    border: none;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  .login-btn:hover:not(:disabled) {
    background: linear-gradient(135deg, #4f63e8, #6366f1);
    box-shadow: 0 4px 16px rgba(60, 80, 224, 0.4);
    transform: translateY(-1px);
  }
  .login-btn:active:not(:disabled) {
    transform: translateY(0);
  }
  .login-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .login-btn :global(.spin-icon) {
    animation: spin 1s linear infinite;
  }

  .version-text {
    margin-top: 24px;
    font-size: 11px;
    color: #475569;
    text-align: center;
    animation: fadeInUp 0.7s ease-out 0.2s both;
  }

  @keyframes fadeInUp {
    from { opacity: 0; transform: translateY(16px); }
    to { opacity: 1; transform: translateY(0); }
  }
  @keyframes float {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-20px); }
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
