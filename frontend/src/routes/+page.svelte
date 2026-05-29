<script lang="ts">
  import { onMount } from 'svelte';
  import {
    ArrowRight,
    CheckCircle2,
    Cpu,
    Gauge,
    Globe,
    LayoutDashboard,
    LineChart,
    Lock,
    ShieldCheck,
    Sparkles,
    Zap
  } from 'lucide-svelte';
  import { api } from '$lib/api';

  let mounted = $state(false);
  let checkingAuth = $state(true);
  let isAuthenticated = $state(false);

  const features = [
    {
      icon: Globe,
      title: 'Unified Provider Routing',
      description: 'Satu endpoint untuk banyak provider/model dengan fallback cerdas dan failover otomatis.'
    },
    {
      icon: Gauge,
      title: 'Low-Latency Performance',
      description: 'Pantau latency, cache hit rate, dan request health dalam dashboard observability real-time.'
    },
    {
      icon: ShieldCheck,
      title: 'Secure by Default',
      description: 'JWT auth, API key controls, audit logs, dan proteksi akses untuk operasi production.'
    },
    {
      icon: Cpu,
      title: 'Format Translation',
      description: 'Bridging format lintas provider tanpa ubah client besar-besaran di layer aplikasi kamu.'
    }
  ];

  const flow = [
    {
      title: 'Connect Providers',
      text: 'Tambah account provider, set credentials, dan aktifkan model discovery.'
    },
    {
      title: 'Configure Routing',
      text: 'Atur priority, fallback chain, dan kebijakan cache sesuai kebutuhan traffic.'
    },
    {
      title: 'Ship with Confidence',
      text: 'Monitor statistik request + logs untuk validasi reliability dan cost efficiency.'
    }
  ];

  const trustStats = [
    { label: 'Response Uptime', value: '99.9%', icon: CheckCircle2 },
    { label: 'Avg Gateway Latency', value: '< 120ms', icon: Zap },
    { label: 'Active Model Paths', value: '40+', icon: Sparkles },
    { label: 'Visibility Coverage', value: 'End-to-end', icon: LineChart }
  ];

  onMount(async () => {
    mounted = true;
    const token = localStorage.getItem('lintasan_token');

    if (!token) {
      checkingAuth = false;
      return;
    }

    try {
      await api.get('/api/auth/me');
      isAuthenticated = true;
    } catch {
      localStorage.removeItem('lintasan_token');
      localStorage.removeItem('lintasan_user');
      isAuthenticated = false;
    } finally {
      checkingAuth = false;
    }
  });

  const primaryHref = $derived(
    checkingAuth ? '/login' : (isAuthenticated ? '/dashboard' : '/login')
  );
  const primaryLabel = $derived(
    checkingAuth ? 'Preparing Workspace...' : (isAuthenticated ? 'Go to Dashboard' : 'Sign in to Dashboard')
  );
</script>

<svelte:head>
  <title>Lintasan — AI Gateway Management</title>
  <meta
    name="description"
    content="Lintasan membantu tim mengelola routing, fallback, observability, dan keamanan AI gateway dalam satu dashboard modern."
  />
</svelte:head>

<div class="landing" class:mounted>
  <div class="noise"></div>
  <div class="aurora aurora-a"></div>
  <div class="aurora aurora-b"></div>

  <header class="topbar">
    <a class="brand" href="/">
      <span class="brand-mark">L</span>
      <span class="brand-text">Lintasan</span>
    </a>
    <div class="topbar-actions">
      <a class="link-muted" href="/dashboard/docs">Docs</a>
      <a class="btn-ghost" href="/login">Login</a>
    </div>
  </header>

  <main>
    <section class="hero section-wrap">
      <div class="hero-copy">
        <p class="kicker">AI Gateway Control Plane</p>
        <h1>Kelola multi-provider LLM routing dengan UI modern, aman, dan siap produksi.</h1>
        <p class="subtitle">
          Dari connection management sampai observability real-time, Lintasan bikin operasi AI stack lebih cepat,
          lebih rapi, dan lebih hemat biaya.
        </p>

        <div class="hero-cta">
          <a class="btn-primary" href={primaryHref} aria-busy={checkingAuth}>
            <LayoutDashboard size={16} />
            {primaryLabel}
            <ArrowRight size={16} />
          </a>
          <a class="btn-soft" href="/dashboard/docs">View Documentation</a>
        </div>

        <div class="chip-row">
          <span class="chip"><Lock size={13} /> JWT Auth</span>
          <span class="chip"><ShieldCheck size={13} /> Secure Routing</span>
          <span class="chip"><Gauge size={13} /> Live Metrics</span>
        </div>
      </div>

      <div class="hero-panel">
        <div class="panel-head">Gateway Snapshot</div>
        <div class="panel-grid">
          <article>
            <span class="label">Traffic Health</span>
            <strong>Stable</strong>
            <small>Request success trend positif</small>
          </article>
          <article>
            <span class="label">Fallback Readiness</span>
            <strong>Enabled</strong>
            <small>Policy multi-path aktif</small>
          </article>
          <article>
            <span class="label">Cache Efficiency</span>
            <strong>Optimized</strong>
            <small>Hit-rate konsisten tinggi</small>
          </article>
          <article>
            <span class="label">Routing Visibility</span>
            <strong>Realtime</strong>
            <small>Logs + analytics terpusat</small>
          </article>
        </div>
      </div>
    </section>

    <section class="section-wrap feature-section">
      <div class="section-title">
        <h2>Built for reliability dan speed</h2>
        <p>Komponen utama untuk operasional AI gateway harian tim engineering dan DevOps.</p>
      </div>
      <div class="feature-grid">
        {#each features as item}
          <article class="feature-card">
            <div class="icon-wrap"><item.icon size={18} /></div>
            <h3>{item.title}</h3>
            <p>{item.description}</p>
          </article>
        {/each}
      </div>
    </section>

    <section class="section-wrap how-section">
      <div class="section-title">
        <h2>How it works</h2>
        <p>3 langkah sederhana untuk produksi yang terukur dan minim friction.</p>
      </div>
      <div class="how-grid">
        {#each flow as step, idx}
          <article class="how-card">
            <span class="step-number">0{idx + 1}</span>
            <h3>{step.title}</h3>
            <p>{step.text}</p>
          </article>
        {/each}
      </div>
    </section>

    <section class="section-wrap trust-strip">
      {#each trustStats as stat}
        <article class="trust-item">
          <div class="trust-icon"><stat.icon size={16} /></div>
          <div>
            <p class="trust-value">{stat.value}</p>
            <p class="trust-label">{stat.label}</p>
          </div>
        </article>
      {/each}
    </section>
  </main>

  <footer class="footer section-wrap">
    <div>
      <strong>Lintasan</strong>
      <p>Observability-first AI gateway management dashboard.</p>
    </div>
    <div class="footer-links">
      <a href="/dashboard/docs">Documentation</a>
      <a href="/login">Login</a>
      <a href={primaryHref}>Open Dashboard</a>
    </div>
  </footer>
</div>

<style>
  .landing {
    min-height: 100vh;
    position: relative;
    background:
      radial-gradient(circle at 15% 20%, rgba(59,130,246,0.16), transparent 38%),
      radial-gradient(circle at 85% 10%, rgba(139,92,246,0.18), transparent 34%),
      linear-gradient(180deg, #070c18 0%, #0b1220 42%, #0f172a 100%);
    color: #e2e8f0;
    overflow: hidden;
    opacity: 0;
    transition: opacity 0.45s ease;
  }

  .landing.mounted { opacity: 1; }

  .noise {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(rgba(148,163,184,0.07) 1px, transparent 1px),
      linear-gradient(90deg, rgba(148,163,184,0.07) 1px, transparent 1px);
    background-size: 44px 44px;
    mask-image: linear-gradient(to bottom, rgba(0,0,0,0.8), transparent 90%);
    pointer-events: none;
  }

  .aurora {
    position: absolute;
    border-radius: 999px;
    filter: blur(88px);
    pointer-events: none;
    opacity: 0.6;
  }

  .aurora-a {
    width: 460px;
    height: 460px;
    top: -180px;
    right: -120px;
    background: rgba(59,130,246,0.42);
    animation: drift 9s ease-in-out infinite;
  }

  .aurora-b {
    width: 420px;
    height: 420px;
    bottom: -170px;
    left: -100px;
    background: rgba(99,102,241,0.35);
    animation: drift 11s ease-in-out infinite reverse;
  }

  .section-wrap {
    width: min(1120px, calc(100% - 48px));
    margin: 0 auto;
  }

  .topbar {
    position: relative;
    z-index: 3;
    width: min(1120px, calc(100% - 48px));
    margin: 0 auto;
    padding: 20px 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .brand {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    text-decoration: none;
  }

  .brand-mark {
    width: 34px;
    height: 34px;
    border-radius: 10px;
    display: grid;
    place-items: center;
    font-size: 14px;
    font-weight: 700;
    color: white;
    background: linear-gradient(135deg, #3c50e0 0%, #6366f1 100%);
    box-shadow: 0 8px 20px rgba(60,80,224,0.4);
  }

  .brand-text {
    color: #f8fafc;
    font-size: 16px;
    font-weight: 700;
    letter-spacing: -0.2px;
  }

  .topbar-actions {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .link-muted {
    text-decoration: none;
    color: #cbd5e1;
    font-size: 13px;
    font-weight: 500;
  }

  .link-muted:hover { color: #f8fafc; }

  .btn-ghost {
    text-decoration: none;
    border: 1px solid rgba(148,163,184,0.25);
    color: #e2e8f0;
    padding: 8px 14px;
    border-radius: 10px;
    font-size: 13px;
    font-weight: 600;
    transition: all 0.2s ease;
    background: rgba(15,23,42,0.55);
  }

  .btn-ghost:hover {
    border-color: rgba(148,163,184,0.45);
    background: rgba(30,41,59,0.75);
  }

  .hero {
    position: relative;
    z-index: 2;
    padding: 44px 0 34px;
    display: grid;
    grid-template-columns: 1.1fr 0.9fr;
    gap: 28px;
    align-items: stretch;
  }

  .kicker {
    margin: 0 0 12px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-size: 11px;
    color: #93c5fd;
    font-weight: 700;
  }

  .hero-copy h1 {
    margin: 0;
    font-size: clamp(34px, 5vw, 50px);
    line-height: 1.1;
    color: #f8fafc;
    letter-spacing: -0.04em;
    max-width: 16ch;
  }

  .subtitle {
    margin: 18px 0 0;
    font-size: 16px;
    line-height: 1.75;
    color: #cbd5e1;
    max-width: 58ch;
  }

  .hero-cta {
    margin-top: 26px;
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }

  .btn-primary,
  .btn-soft {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    border-radius: 12px;
    padding: 11px 18px;
    text-decoration: none;
    font-size: 14px;
    font-weight: 600;
    transition: all 0.2s ease;
  }

  .btn-primary {
    color: #fff;
    background: linear-gradient(135deg, #3c50e0 0%, #4f63e8 100%);
    box-shadow: 0 12px 24px rgba(60,80,224,0.3);
  }

  .btn-primary:hover {
    transform: translateY(-1px);
    box-shadow: 0 16px 28px rgba(60,80,224,0.34);
  }

  .btn-soft {
    color: #e2e8f0;
    border: 1px solid rgba(148,163,184,0.24);
    background: rgba(15,23,42,0.55);
  }

  .btn-soft:hover {
    background: rgba(30,41,59,0.72);
    border-color: rgba(148,163,184,0.42);
  }

  .chip-row {
    margin-top: 18px;
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }

  .chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 8px 11px;
    border-radius: 999px;
    font-size: 12px;
    color: #bfdbfe;
    border: 1px solid rgba(96,165,250,0.26);
    background: rgba(30,64,175,0.18);
  }

  .hero-panel {
    border-radius: 18px;
    border: 1px solid rgba(148,163,184,0.2);
    background: rgba(15,23,42,0.72);
    backdrop-filter: blur(16px);
    padding: 20px;
    box-shadow: 0 22px 48px rgba(2,6,23,0.38);
  }

  .panel-head {
    font-size: 12px;
    font-weight: 700;
    color: #93c5fd;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    margin-bottom: 12px;
  }

  .panel-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }

  .panel-grid article {
    border-radius: 12px;
    padding: 13px;
    border: 1px solid rgba(148,163,184,0.18);
    background: rgba(30,41,59,0.56);
  }

  .panel-grid .label {
    font-size: 11px;
    color: #94a3b8;
    display: block;
    margin-bottom: 5px;
  }

  .panel-grid strong {
    display: block;
    color: #f8fafc;
    font-size: 16px;
    letter-spacing: -0.02em;
  }

  .panel-grid small {
    display: block;
    margin-top: 3px;
    font-size: 12px;
    color: #cbd5e1;
  }

  .feature-section,
  .how-section {
    margin-top: 48px;
  }

  .section-title h2 {
    margin: 0;
    font-size: 29px;
    line-height: 1.2;
    color: #f8fafc;
    letter-spacing: -0.03em;
  }

  .section-title p {
    margin: 10px 0 0;
    color: #cbd5e1;
    font-size: 14px;
    line-height: 1.65;
  }

  .feature-grid {
    margin-top: 18px;
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
  }

  .feature-card {
    border-radius: 14px;
    padding: 16px;
    border: 1px solid rgba(148,163,184,0.17);
    background: linear-gradient(180deg, rgba(30,41,59,0.58) 0%, rgba(15,23,42,0.78) 100%);
    transition: transform 0.2s ease, border-color 0.2s ease;
  }

  .feature-card:hover {
    transform: translateY(-2px);
    border-color: rgba(96,165,250,0.35);
  }

  .icon-wrap {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    display: grid;
    place-items: center;
    color: #bfdbfe;
    background: rgba(30,64,175,0.28);
    border: 1px solid rgba(96,165,250,0.25);
  }

  .feature-card h3 {
    margin: 12px 0 8px;
    color: #f1f5f9;
    font-size: 15px;
    letter-spacing: -0.02em;
  }

  .feature-card p {
    margin: 0;
    font-size: 13px;
    line-height: 1.65;
    color: #cbd5e1;
  }

  .how-grid {
    margin-top: 18px;
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
  }

  .how-card {
    border-radius: 14px;
    padding: 16px;
    border: 1px solid rgba(148,163,184,0.17);
    background: rgba(15,23,42,0.72);
  }

  .step-number {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 36px;
    height: 26px;
    padding: 0 10px;
    border-radius: 999px;
    font-size: 11px;
    font-weight: 700;
    color: #bfdbfe;
    border: 1px solid rgba(96,165,250,0.32);
    background: rgba(30,64,175,0.2);
  }

  .how-card h3 {
    margin: 12px 0 8px;
    color: #f1f5f9;
    font-size: 16px;
  }

  .how-card p {
    margin: 0;
    color: #cbd5e1;
    font-size: 13px;
    line-height: 1.65;
  }

  .trust-strip {
    margin-top: 44px;
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
    padding: 14px;
    border-radius: 14px;
    border: 1px solid rgba(148,163,184,0.18);
    background: rgba(15,23,42,0.7);
  }

  .trust-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px;
    border-radius: 10px;
    background: rgba(30,41,59,0.55);
    border: 1px solid rgba(148,163,184,0.14);
  }

  .trust-icon {
    width: 31px;
    height: 31px;
    border-radius: 9px;
    display: grid;
    place-items: center;
    color: #bfdbfe;
    background: rgba(30,64,175,0.22);
    border: 1px solid rgba(96,165,250,0.22);
  }

  .trust-value {
    margin: 0;
    color: #f8fafc;
    font-size: 15px;
    font-weight: 700;
    line-height: 1.2;
  }

  .trust-label {
    margin: 2px 0 0;
    color: #94a3b8;
    font-size: 11px;
  }

  .footer {
    margin-top: 44px;
    padding: 24px 0 34px;
    border-top: 1px solid rgba(148,163,184,0.16);
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 18px;
  }

  .footer strong {
    color: #f8fafc;
    font-size: 15px;
  }

  .footer p {
    margin: 7px 0 0;
    color: #94a3b8;
    font-size: 12px;
  }

  .footer-links {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .footer-links a {
    text-decoration: none;
    color: #cbd5e1;
    font-size: 12px;
    font-weight: 600;
  }

  .footer-links a:hover { color: #f8fafc; }

  @media (max-width: 1080px) {
    .hero { grid-template-columns: 1fr; }
    .hero-copy h1 { max-width: 20ch; }
    .feature-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
    .how-grid { grid-template-columns: 1fr; }
    .trust-strip { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  }

  @media (max-width: 640px) {
    .section-wrap,
    .topbar {
      width: calc(100% - 28px);
    }

    .topbar { padding: 14px 0; }
    .brand-text { font-size: 15px; }
    .hero { padding-top: 30px; }

    .hero-copy h1 {
      font-size: 31px;
      max-width: 100%;
    }

    .subtitle { font-size: 15px; }
    .hero-cta { flex-direction: column; }
    .btn-primary, .btn-soft { width: 100%; justify-content: center; }

    .panel-grid { grid-template-columns: 1fr; }
    .feature-grid { grid-template-columns: 1fr; }
    .trust-strip { grid-template-columns: 1fr; }

    .footer {
      align-items: flex-start;
      flex-direction: column;
    }
  }

  @keyframes drift {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-16px); }
  }
</style>
