<script lang="ts">
  import { onMount } from 'svelte';
  import {
    ArrowRight, Gauge, Globe, LayoutDashboard,
    ShieldCheck, Sparkles, Zap
  } from 'lucide-svelte';
  import { api } from '$lib/api';

  let mounted = $state(false);
  let checkingAuth = $state(true);
  let isAuthenticated = $state(false);

  const features = [
    {
      icon: Globe, title: 'Unified Provider Routing',
      desc: 'Satu endpoint untuk banyak provider dengan fallback cerdas dan failover otomatis.'
    },
    {
      icon: Gauge, title: 'Low-Latency Performance',
      desc: 'Pantau latency, cache hit rate, dan request health dalam dashboard real-time.'
    },
    {
      icon: ShieldCheck, title: 'Secure by Default',
      desc: 'JWT auth, API key controls, audit logs, dan proteksi untuk operasi production.'
    },
    {
      icon: Zap, title: 'Smart Caching',
      desc: 'Semantic cache mengurangi beban provider dan mempercepat response hingga 10x.'
    },
    {
      icon: Sparkles, title: 'Format Translation',
      desc: 'Bridging format lintas provider tanpa ubah client di layer aplikasi.'
    },
    {
      icon: LayoutDashboard, title: 'Observability Dashboard',
      desc: 'Logs, analytics, dan monitoring terpusat untuk seluruh AI traffic.'
    }
  ];

  const steps = [
    { step: '01', title: 'Connect Providers', desc: 'Tambah credentials dan aktifkan model discovery.' },
    { step: '02', title: 'Configure Routing', desc: 'Atur priority, fallback chain, dan kebijakan cache.' },
    { step: '03', title: 'Ship with Confidence', desc: 'Monitor statistik dan logs untuk validasi production.' }
  ];

  onMount(async () => {
    mounted = true;
    const token = localStorage.getItem('lintasan_token');
    if (!token) { checkingAuth = false; return; }
    try {
      await api.get('/api/auth/me');
      isAuthenticated = true;
    } catch {
      localStorage.removeItem('lintasan_token');
      localStorage.removeItem('lintasan_user');
    } finally { checkingAuth = false; }
  });

  const ctaHref = $derived(checkingAuth ? '/login' : (isAuthenticated ? '/dashboard' : '/login'));
  const ctaLabel = $derived(checkingAuth ? 'Loading...' : (isAuthenticated ? 'Go to Dashboard' : 'Get Started'));
</script>

<svelte:head>
  <title>Lintasan — AI Gateway Management</title>
  <meta name="description" content="Kelola multi-provider LLM routing dengan UI modern, aman, dan siap produksi." />
</svelte:head>

<div class="landing" class:mounted>
  <header class="topbar">
    <a class="brand" href="/">
      <span class="brand-mark">L</span>
      <span class="brand-name">Lintasan</span>
    </a>
    <nav class="nav-links">
      <a href="/login" class="nav-link">Sign In</a>
      <a href={ctaHref} class="btn-cta">
        {ctaLabel}
        <ArrowRight size={14} />
      </a>
    </nav>
  </header>

  <main>
    <section class="hero">
      <p class="hero-kicker">AI Gateway Control Plane</p>
      <h1 class="hero-title">Route smarter.<br />Ship faster.</h1>
      <p class="hero-sub">
        Lintasan mengelola semua koneksi LLM provider kamu — routing, fallback,
        caching, dan observability — dalam satu dashboard yang bersih dan efisien.
      </p>
      <a href={ctaHref} class="btn-hero">
        {ctaLabel}
        <ArrowRight size={16} />
      </a>
    </section>

    <section class="features">
      <h2 class="section-title">Everything you need</h2>
      <p class="section-sub">Tools lengkap untuk operasional AI engineering harian.</p>
      <div class="feature-grid">
        {#each features as f}
          <article class="feature-card">
            <div class="feature-icon"><f.icon size={22} stroke-width={1.5} /></div>
            <h3>{f.title}</h3>
            <p>{f.desc}</p>
          </article>
        {/each}
      </div>
    </section>

    <section class="how">
      <h2 class="section-title">How it works</h2>
      <p class="section-sub">Tiga langkah sederhana untuk mulai production.</p>
      <div class="how-grid">
        {#each steps as s}
          <article class="how-card">
            <span class="step-num">{s.step}</span>
            <h3>{s.title}</h3>
            <p>{s.desc}</p>
          </article>
        {/each}
      </div>
    </section>
  </main>

  <footer class="footer">
    <div class="footer-brand">
      <strong>Lintasan</strong>
      <span>AI Gateway Management</span>
    </div>
    <div class="footer-links">
      <a href="/login">Sign In</a>
      <a href="/dashboard/docs">Documentation</a>
      <a href="https://github.com/sans-haji/lintasan" target="_blank" rel="noopener noreferrer">GitHub</a>
    </div>
  </footer>
</div>

<style>
  .landing {
    min-height: 100vh;
    background: #ffffff;
    color: #1e293b;
    font-family: 'Inter', system-ui, -apple-system, sans-serif;
    opacity: 0;
    transition: opacity 0.3s ease;
  }
  .landing.mounted { opacity: 1; }

  .topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px 32px;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    text-decoration: none;
  }
  .brand-mark {
    width: 36px; height: 36px;
    border-radius: 10px;
    background: #4f46e5;
    color: #fff;
    display: grid;
    place-items: center;
    font-weight: 700;
    font-size: 15px;
  }
  .brand-name {
    font-size: 18px;
    font-weight: 700;
    color: #1e293b;
    letter-spacing: -0.3px;
  }

  .nav-links { display: flex; align-items: center; gap: 16px; }
  .nav-link {
    text-decoration: none;
    color: #64748b;
    font-size: 14px;
    font-weight: 500;
    padding: 8px 4px;
  }
  .nav-link:hover { color: #1e293b; }

  .btn-cta {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 9px 18px;
    background: #4f46e5;
    color: #fff;
    border-radius: 10px;
    font-size: 13px;
    font-weight: 600;
    text-decoration: none;
    transition: background 0.15s;
  }
  .btn-cta:hover { background: #4338ca; }

  .hero {
    max-width: 720px;
    margin: 0 auto;
    padding: 80px 32px 64px;
    text-align: center;
  }
  .hero-kicker {
    margin: 0 0 12px;
    font-size: 13px;
    font-weight: 700;
    letter-spacing: 0.06em;
    color: #4f46e5;
    text-transform: uppercase;
  }
  .hero-title {
    margin: 0 0 20px;
    font-size: clamp(40px, 7vw, 64px);
    line-height: 1.1;
    letter-spacing: -0.04em;
    color: #0f172a;
  }
  .hero-sub {
    margin: 0 auto 32px;
    max-width: 520px;
    font-size: 17px;
    line-height: 1.65;
    color: #64748b;
  }
  .btn-hero {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 14px 28px;
    background: #4f46e5;
    color: #fff;
    border-radius: 12px;
    font-size: 15px;
    font-weight: 600;
    text-decoration: none;
    transition: background 0.15s, transform 0.15s;
  }
  .btn-hero:hover { background: #4338ca; transform: translateY(-1px); }

  .section-title {
    margin: 0 0 8px;
    font-size: 28px;
    font-weight: 700;
    letter-spacing: -0.03em;
    color: #0f172a;
  }
  .section-sub {
    margin: 0 0 48px;
    font-size: 16px;
    color: #64748b;
  }

  .features {
    max-width: 1100px;
    margin: 0 auto;
    padding: 48px 32px 64px;
  }
  .feature-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 20px;
  }
  .feature-card {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    padding: 28px 24px;
    transition: border-color 0.2s, box-shadow 0.2s;
  }
  .feature-card:hover {
    border-color: #c7d2fe;
    box-shadow: 0 4px 16px rgba(79, 70, 229, 0.08);
  }
  .feature-icon {
    width: 44px; height: 44px;
    border-radius: 12px;
    background: #eef2ff;
    color: #4f46e5;
    display: grid;
    place-items: center;
    margin-bottom: 16px;
  }
  .feature-card h3 {
    margin: 0 0 8px;
    font-size: 16px;
    font-weight: 600;
    color: #1e293b;
  }
  .feature-card p {
    margin: 0;
    font-size: 14px;
    line-height: 1.6;
    color: #64748b;
  }

  .how {
    max-width: 1100px;
    margin: 0 auto;
    padding: 48px 32px 80px;
    text-align: center;
  }
  .how-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 20px;
  }
  .how-card {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    padding: 32px 24px;
    text-align: left;
  }
  .step-num {
    display: inline-block;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.04em;
    color: #4f46e5;
    background: #eef2ff;
    padding: 4px 10px;
    border-radius: 6px;
    margin-bottom: 16px;
  }
  .how-card h3 {
    margin: 0 0 8px;
    font-size: 16px;
    font-weight: 600;
    color: #1e293b;
  }
  .how-card p {
    margin: 0;
    font-size: 14px;
    line-height: 1.6;
    color: #64748b;
  }

  .footer {
    max-width: 1100px;
    margin: 0 auto;
    padding: 32px;
    border-top: 1px solid #e2e8f0;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 16px;
  }
  .footer-brand strong {
    display: block;
    font-size: 15px;
    color: #1e293b;
  }
  .footer-brand span {
    font-size: 13px;
    color: #94a3b8;
  }
  .footer-links {
    display: flex;
    gap: 20px;
  }
  .footer-links a {
    text-decoration: none;
    font-size: 13px;
    color: #64748b;
  }
  .footer-links a:hover { color: #1e293b; }

  @media (max-width: 640px) {
    .topbar { padding: 16px 20px; }
    .hero { padding: 48px 20px 40px; }
    .hero-title { font-size: 36px; }
    .features, .how { padding: 32px 20px 48px; }
    .feature-grid { grid-template-columns: 1fr; }
    .how-grid { grid-template-columns: 1fr; }
    .footer { flex-direction: column; align-items: flex-start; }
  }
</style>
