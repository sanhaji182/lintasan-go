# 🇮🇩 Lintasan

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Version](https://img.shields.io/badge/version-v0.24.2-blue)](https://github.com/sanhaji182/lintasan/releases)
[![Tests](https://img.shields.io/badge/tests-700+-success)](.)

> **Setiap Koneksi Punya Jalannya** — Jalur cerdas yang menghubungkan manusia, AI, dan sistem dalam satu aliran terintegrasi.

> ⚡ 35x lebih ringan dari Node.js v1 · single binary ~24MB · 700+ tests · 118+ provider presets · Go (v0.24.2) + SvelteKit Embedded SPA

---

## 🇬🇧 Lintasan

> **Every Connection Has Its Path** — An intelligent pathway connecting humans, AI, and systems in one integrated flow.

> ⚡ 35x lighter than Node.js v1 · single ~24MB binary · 700+ tests · 118+ provider presets · Go (v0.24.2) + SvelteKit Embedded SPA

---

## 🌐 Filosofi / Philosophy

<details open>
<summary>🇮🇩 Bahasa Indonesia</summary>

"Lintasan adalah jalur tempat koneksi, kecerdasan, dan kemungkinan bergerak."

Di dunia modern, AI bukan hanya tentang model. AI adalah tentang bagaimana sistem saling terhubung, bagaimana request menemukan jalur terbaik, dan bagaimana manusia berinteraksi dengan teknologi secara efisien.

Lintasan hadir sebagai jalur cerdas yang menghubungkan manusia, AI, dan sistem dalam satu aliran yang terintegrasi. Satu endpoint untuk semua provider AI — tidak perlu lagi berganti-ganti SDK atau mengelola banyak API key di berbagai tempat.

</details>

<details>
<summary>🇬🇧 English</summary>

"Lintasan is the path where connections, intelligence, and possibilities move."

In the modern world, AI isn't just about models. It's about how systems connect, how requests find their best path, and how humans interact with technology efficiently.

Lintasan exists as an intelligent pathway connecting humans, AI, and systems in one integrated flow. One endpoint for all AI providers — no more switching SDKs or managing multiple API keys across different places.

</details>

---

## 📖 Daftar Isi / Table of Contents

- [Apa itu Lintasan? / What is Lintasan?](#-apa-itu-lintasan--what-is-lintasan)
- [Kenapa Go? / Why Go?](#-kenapa-go--why-go)
- [Tech Stack](#-tech-stack)
- [Fitur Utama / Key Features](#-fitur-utama--key-features)
- [Arsitektur / Architecture](#-arsitektur--architecture)
- [Quick Start](#-quick-start)
- [Instalasi / Installation](#-instalasi--installation)
- [Konfigurasi / Configuration](#-konfigurasi--configuration)
- [API Usage](#-api-usage)
- [Dashboard](#-dashboard)
- [Provider Presets](#-provider-presets)
- [Struktur Project / Project Structure](#-struktur-project--project-structure)
- [Development](#-development)
- [Testing](#-testing)
- [Deployment](#-deployment)
- [Benchmark](#-benchmark)
- [Migration dari Node.js v1](#-migration-dari-nodejs-v1)
- [Contributing](#-contributing)
- [Lisensi / License](#-lisensi--license)

---

## ❓ Apa itu Lintasan? / What is Lintasan?

<details open>
<summary>🇮🇩 Bahasa Indonesia</summary>

Lintasan adalah **LLM proxy gateway** dengan 40+ fitur optimasi. Satu endpoint OpenAI-compatible untuk semua provider AI — routing cerdas, embedded cache, dual-mode CommandCode, plugin system, dan penghematan token otomatis.

**Masalah yang diselesaikan:**
- 🔀 **Multi-provider complexity** — ganti provider = ganti SDK, ganti API key, ganti format
- 💸 **No cost visibility** — tidak tahu berapa token terpakai per model/provider
- 🔄 **No failover** — satu provider down, semua request gagal
- 🔐 **Key management chaos** — API key tersebar di mana-mana
- 📊 **No observability** — tidak bisa lihat usage, latency, error rate

**Solusi Lintasan:**
- Satu endpoint → semua provider (OpenAI, Anthropic, DeepSeek, Gemini, Groq, dll)
- Satu API key → autentikasi terpusat
- Smart routing → request otomatis ke provider terbaik
- Circuit breaker → provider gagal auto-disable
- Dashboard → monitoring real-time 17 halaman
- Plugin system → extensible tanpa ubah core

</details>

<details>
<summary>🇬🇧 English</summary>

Lintasan is an **LLM proxy gateway** with 40+ optimization features. One OpenAI-compatible endpoint for all AI providers — smart routing, embedded cache, dual-mode CommandCode, plugin system, and automatic token savings.

**Problems solved:**
- 🔀 **Multi-provider complexity** — switching providers means switching SDKs, API keys, and formats
- 💸 **No cost visibility** — can't see token usage per model/provider
- 🔄 **No failover** — one provider down, all requests fail
- 🔐 **Key management chaos** — API keys scattered everywhere
- 📊 **No observability** — can't see usage, latency, error rates

**Lintasan's solution:**
- One endpoint → all providers (OpenAI, Anthropic, DeepSeek, Gemini, Groq, etc.)
- One API key → centralized authentication
- Smart routing → requests automatically go to the best provider
- Circuit breaker → failing providers auto-disable
- Dashboard → 17-page real-time monitoring
- Plugin system → extensible without core changes

</details>

---

## 💪 Kenapa Go? / Why Go?

| Metric | Node.js (legacy) | Go (current) | Improvement |
|--------|------------------|--------------|-------------|
| **RAM** | ~500MB | ~14MB | **35x lebih hemat** |
| **Binary size** | 513MB (node_modules) | 24MB (single file) | **21x lebih kecil** |
| **Startup** | 3-5 detik | <50ms | **60-100x lebih cepat** |
| **Concurrent req/s** | ~10,000 | ~50,000+ | **5x throughput** |
| **Dependencies** | 800+ npm packages | 1 (go-sqlite3) | **800x lebih sedikit** |
| **Tests** | Manual | 700+ / 35 packages | **Automated** |
| **Provider presets** | 27 | 118 | **4x lebih banyak** |
| **Deployment** | Docker + npm install | `scp` 24MB binary | **Zero setup** |

---

## 🛠 Tech Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Backend** | Go 1.22+ | HTTP server, routing, proxy, streaming |
| **Database** | SQLite (go-sqlite3) | Embedded, zero config, single-file |
| **Frontend** | SvelteKit 5 + TypeScript | Dashboard SPA, client-rendered, **embedded into the Go binary via go:embed** |
| **Styling** | Tailwind CSS v4 + CSS variables | Dark/light mode, responsive |
| **CLI** | Cobra | `start`, `setup`, `mitm`, `version` |
| **Testing** | Go standard library | 581 tests, 39 packages |
| **Deployment** | **Single self-contained binary** + systemd | UI + API in one executable; **no Node.js required**. Pure proxy |

---

## ✨ Fitur Utama / Key Features

> **⚠️ Kebijakan Legal & Etika / Legal & Ethics Policy**
>
> **🇮🇩** Lintasan didesain untuk menggunakan **API resmi yang sah** (Legal API). Kami secara tegas **tidak melakukan/mendukung Reverse Engineering** terhadap endpoint internal IDE komersial. Lintasan berfokus pada integrasi provider API resmi dan Experimental Provider (ACP).
>
> **🇬🇧** Lintasan is designed to use **legitimate, official APIs**. We strictly do **not support reverse engineering** of commercial IDE internal endpoints. Lintasan focuses on official API provider integrations and Experimental Provider pipelines (ACP).

<details open>
<summary>🇮🇩 Fitur</summary>

| # | Fitur | Deskripsi |
|---|-------|-----------|
| 1 | **Multi-Provider Proxy** | Satu endpoint untuk 10+ provider LLM |
| 2 | **Smart Routing** | Multi-stage: regex → header → model → fallback |
| 3 | **113 Provider Presets** | Semua provider LiteLLM siap pakai |
| 4 | **Connection Management** | Add/test/sync/delete connections + auto discovery |
| 5 | **Model Discovery** | Auto-fetch models dari provider /models endpoint |
| 6 | **Provider Test** | Real-time latency + model count testing |
| 7 | **Streaming (SSE)** | Full Server-Sent Events untuk streaming |
| 8 | **Fallback Chains** | Multi-level fallback per model |
| 9 | **Circuit Breaker** | Auto-disable provider yang gagal |
| 10 | **Request Logging** | Complete request/response logging |
| 11 | **Analytics** | Real-time metrics: latency, tokens, throughput |
| 12 | **Combo System** | Pre-configured model+provider bundles |
| 13 | **Load Balancer** | Model-aware weighted load balancing |
| 14 | **Plugin System** | Plugin store + auto-registration |
| 15 | **Vector Memory** | Pluggable embedder dengan SQLite default |
| 16 | **Web Search** | Augment chat dengan live web results |
| 17 | **OAuth Integration** | Provider OAuth flow support |
| 18 | **Image Generation** | Proxy ke DALL-E / Stable Diffusion |
| 19 | **Audio (TTS + STT)** | Speech + transcription via OpenAI API |
| 20 | **Token Budgeting** | Per-key daily/monthly limits |
| 21 | **Cost Tracking** | Real-time cost tracking per request |
| 22 | **API Keys** | Key management + usage tracking |
| 23 | **Teams & Users** | Multi-user access control |
| 24 | **Webhooks** | Event-driven webhook system |
| 25 | **Backup & Export** | Database backup + disaster recovery |
| 26 | **Dashboard** | 17-page interactive SvelteKit dashboard |
| 27 | **Playground** | Built-in API test console |
| 28 | **CORS** | Built-in — use from any browser app |
| 29 | **Zero Config** | SQLite embedded — no setup required |
| 30 | **CLI (Cobra)** | `start`, `setup`, `mitm`, `version` |
| 31 | **MITM Bridge** | Optional HTTPS bridge untuk LocalAI/LM Studio |
| 32 | **Experimental Mode** | Cohort Provider ACP execution environment |
| 33 | **Vector Search** | In-memory TF-IDF + auto context injection |
| 34 | **Credential Vault** | AES-256-GCM UI Credential Dashboard |
| 35 | **cURL Import** | Quick-add connections from cURL commands |
| 36 | **Metrics Endpoint** | Exposes `/metrics` endpoint and Dashboard Observability UI |

</details>

<details>
<summary>🇬🇧 Features</summary>

| # | Feature | Description |
|---|---------|-------------|
| 1 | **Multi-Provider Proxy** | One endpoint for 10+ LLM providers |
| 2 | **Smart Routing** | Multi-stage: regex → header → model → fallback |
| 3 | **113 Provider Presets** | All LiteLLM providers ready to use |
| 4 | **Connection Management** | Add/test/sync/delete connections + auto discovery |
| 5 | **Model Discovery** | Auto-fetch models from provider /models endpoint |
| 6 | **Provider Test** | Real-time latency + model count testing |
| 7 | **Streaming (SSE)** | Full Server-Sent Events for streaming |
| 8 | **Fallback Chains** | Multi-level fallback per model |
| 9 | **Circuit Breaker** | Auto-disable failing providers |
| 10 | **Request Logging** | Complete request/response logging |
| 11 | **Analytics** | Real-time metrics: latency, tokens, throughput |
| 12 | **Combo System** | Pre-configured model+provider bundles |
| 13 | **Load Balancer** | Model-aware weighted load balancing |
| 14 | **Plugin System** | Plugin store + auto-registration |
| 15 | **Vector Memory** | Pluggable embedder with SQLite default |
| 16 | **Web Search** | Augment chat with live web results |
| 17 | **OAuth Integration** | Provider OAuth flow support |
| 18 | **Image Generation** | Proxy to DALL-E / Stable Diffusion |
| 19 | **Audio (TTS + STT)** | Speech + transcription via OpenAI API |
| 20 | **Token Budgeting** | Per-key daily/monthly limits |
| 21 | **Cost Tracking** | Real-time cost tracking per request |
| 22 | **API Keys** | Key management + usage tracking |
| 23 | **Teams & Users** | Multi-user access control |
| 24 | **Webhooks** | Event-driven webhook system |
| 25 | **Backup & Export** | Database backup + disaster recovery |
| 26 | **Dashboard** | 17-page interactive SvelteKit dashboard |
| 27 | **Playground** | Built-in API test console |
| 28 | **CORS** | Built-in — use from any browser app |
| 29 | **Zero Config** | SQLite embedded — no setup required |
| 30 | **CLI (Cobra)** | `start`, `setup`, `mitm`, `version` |
| 31 | **MITM Bridge** | Optional HTTPS bridge for LocalAI/LM Studio |
| 32 | **Experimental Mode** | Cohort Provider ACP execution environment |
| 33 | **Vector Search** | In-memory TF-IDF + auto context injection |
| 34 | **Credential Vault** | AES-256-GCM UI Credential Dashboard |
| 35 | **cURL Import** | Quick-add connections from cURL commands |
| 36 | **Metrics Endpoint** | Exposes `/metrics` endpoint and Dashboard Observability UI |

</details>

---

## 🏗 Arsitektur / Architecture

```
Client (App / Agent / curl / IDE)
        │
    ▼             ▼
┌─────────────────────────────────────────────────────┐
│ Nginx (SSL Termination) — lintasan.sans.biz.id      │
│   * All traffic proxies to Go:20180                 │
└─────────────────────────┬───────────────────────────┘
                          │
         ┌────────────────▼──────────────────┐
         │ Go Backend :20180 (lintasan start)│
         │ ── Serves BOTH API & UI ──        │
         │ • Embedded SPA UI (go:embed)      │
         │ • OpenAI-compatible LLM proxy     │
         └──────────────┬────────────────────┘
     │
     │  ┌─────────────────────────┐
     │  │  API Gateway            │
     │  │  /v1/chat/completions   │
     │  │  /api/connections/import-curl │
     │  │  /v1/embeddings         │
     │  │  /v1/images/generations │
     │  │  /v1/audio/*            │
     │  │  /metrics               │
     │  │  /v1/models             │
     │  │  /v1/memory/*           │
     │  └──────────┬──────────────┘
     │             │
     │  ┌──────────▼──────────────┐
     │  │  Smart Router           │
     │  │  1. Header-based        │
     │  │  2. Model name match    │
     │  │  3. Load-balanced pick  │
     │  │  4. Priority sort       │
     │  │  5. Fallback chain      │
     │  └──────────┬──────────────┘
     │             │
     │  ┌──────────▼──────────────┐
     │  │  Optimization Pipeline  │
     │  │  • Circuit Breaker      │
     │  │  • Settings Cache (5s)  │
     │  │  • Request Logging      │
     │  │  • Cost Tracking        │
     │  │  • Vector Context Inject│
     │  │  • RTK Compressor       │
     │  └──────────┬──────────────┘
     │             │
     │  ┌──────────▼──────────────┐
     │  │  Provider Dispatcher    │
     │  │  + HTTP/1.1 keep-alive  │
     │  │  + Connection pooling   │
     │  │  + Request translation  │
     │  │  + SSE streaming        │
     │  └──────────┬──────────────┘
     │             │
     └─────────────┼───────────────────
                   │
         ┌─────────┼─────────┬──────────┐
         ▼         ▼         ▼          ▼
    ┌────────┐ ┌──────┐ ┌──────┐ ┌──────────┐
    │ OpenAI │ │Gemini│ │Groq  │ │Sumopod   │ ...118 providers
    └────────┘ └──────┘ └──────┘ └──────────┘
```

---

## 🚀 Quick Start

The dashboard UI is **embedded inside the binary**, so one executable serves the full app (UI + API) on `:20180` — no Node, no nginx.

```bash
# Download the pre-built binary (Linux x86_64) and run
# curl -L -o lintasan https://github.com/sanhaji182/lintasan/releases/latest/download/lintasan-linux-amd64
# chmod +x lintasan
# ./lintasan start

# Dashboard → http://localhost:20180/   (redirects to /dashboard)
# API       → http://localhost:20180/v1/chat/completions
```

First run seeds an admin account and prints a one-time password to the console — log in, rotate it, and set a master key to finish setup.

---

## 📦 Instalasi / Installation

<details open>
<summary>🇮🇩 3 Cara Install</summary>

### Opsi 1: Binary Pre-built (Recommended)
Satu binary, dashboard sudah termasuk di dalamnya.
```bash
curl -L -o lintasan https://github.com/sanhaji182/lintasan/releases/latest/download/lintasan-linux-amd64
chmod +x lintasan
./lintasan start
```

### Opsi 2: Build dari Source
Butuh **Go 1.22+** dan **Node 20+** (untuk build dashboard). `make build` meng-compile frontend SvelteKit jadi SPA statis, meng-embed-nya ke binary Go, lalu compile binary tunggal.
```bash
git clone https://github.com/sanhaji182/lintasan.git
cd lintasan
make build       # frontend → embed → ./lintasan
./lintasan start
```
> Tanpa Node? `go build -o lintasan ./cmd/lintasan` tetap jalan, tapi menghasilkan server **API-only** (tanpa UI dashboard).

### Opsi 3: Docker (single container)
```bash
git clone https://github.com/sanhaji182/lintasan.git
cd lintasan
LINTASAN_MASTER_KEY=$(openssl rand -hex 32) docker compose up --build
# UI + API → http://localhost:20180
```

### CLI Commands
```bash
./lintasan start      # Start server (UI + API, default :20180)
./lintasan setup      # Initialize database
./lintasan mitm start # Start MITM HTTPS bridge (optional)
./lintasan version    # Show version
./lintasan help       # All commands

# Custom port
PORT=8080 ./lintasan start
```

</details>

<details>
<summary>🇬🇧 3 Installation Methods</summary>

### Option 1: Pre-built Binary (Recommended)
One binary, dashboard included.
```bash
curl -L -o lintasan https://github.com/sanhaji182/lintasan/releases/latest/download/lintasan-linux-amd64
chmod +x lintasan
./lintasan start
```

### Option 2: Build from Source
Requires **Go 1.22+** and **Node 20+** (to build the dashboard). `make build` compiles the SvelteKit frontend into a static SPA, embeds it into the Go binary, and compiles a single executable.
```bash
git clone https://github.com/sanhaji182/lintasan.git
cd lintasan
make build       # frontend → embed → ./lintasan
./lintasan start
```
> No Node? `go build -o lintasan ./cmd/lintasan` still works but produces an **API-only** server (no dashboard UI).

### Option 3: Docker (single container)
```bash
git clone https://github.com/sanhaji182/lintasan.git
cd lintasan
LINTASAN_MASTER_KEY=$(openssl rand -hex 32) docker compose up --build
# UI + API → http://localhost:20180
```

### CLI Commands
```bash
./lintasan start      # Start server (UI + API, default :20180)
./lintasan setup      # Initialize database
./lintasan mitm start # Start MITM HTTPS bridge (optional)
./lintasan version    # Show version
./lintasan help       # All commands

# Custom port
PORT=8080 ./lintasan start
```

</details>

---

## ⚙ Konfigurasi / Configuration

<details open>
<summary>🇮🇩 Environment Variables</summary>

| Variable | Default | Keterangan |
|----------|---------|------------|
| `PORT` | `20180` | Port server utama |
| `LINTASAN_DATA_DIR` | `./data` | Direktori data (DB, logs) |
| `LINTASAN_MASTER_KEY` | auto-generated | Master API key |
| `MITM_PORT` | `8443` | MITM bridge port |

Tidak perlu `.env` file — set env vars atau gunakan default. Database auto-create saat pertama run.

</details>

<details>
<summary>🇬🇧 Environment Variables</summary>

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `20180` | Main server port |
| `LINTASAN_DATA_DIR` | `./data` | Data directory (DB, logs) |
| `LINTASAN_MASTER_KEY` | auto-generated | Master API key |
| `MITM_PORT` | `8443` | MITM bridge port |

No `.env` file needed — just set env vars or use defaults. Database auto-creates on first run.

</details>

---

## 📡 API Usage

```bash
# Chat completion (OpenAI-compatible)
curl http://localhost:20180/v1/chat/completions \
  -H "Authorization: Bearer YOUR_MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'

# List models
curl http://localhost:20180/v1/models \
  -H "Authorization: Bearer YOUR_MASTER_KEY"

# Embeddings
curl http://localhost:20180/v1/embeddings \
  -H "Authorization: Bearer YOUR_MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "text-embedding-3-small", "input": "Hello world"}'

# Image generation
curl http://localhost:20180/v1/images/generations \
  -H "Authorization: Bearer YOUR_MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "dall-e-3", "prompt": "A sunset over mountains"}'

# Text-to-speech
curl http://localhost:20180/v1/audio/speech \
  -H "Authorization: Bearer YOUR_MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "tts-1", "input": "Hello world", "voice": "alloy"}'

# Speech-to-text
curl http://localhost:20180/v1/audio/transcriptions \
  -H "Authorization: Bearer YOUR_MASTER_KEY" \
  -F "file=@audio.mp3" -F "model=whisper-1"

# Provider presets
curl http://localhost:20180/api/providers/presets

# Connection test
curl -X POST http://localhost:20180/api/connections/test \
  -H "Content-Type: application/json" \
  -d '{"base_url": "https://api.openai.com/v1", "api_key": "sk-..."}'

# Vector memory search
curl "http://localhost:20180/v1/memory/search?q=hello&limit=5" \
  -H "Authorization: Bearer ***
  
# Import curl via API Connection Creation
curl -X POST "http://localhost:20180/api/connections/import-curl" \
  -H "Authorization: Bearer ***

# Web search augmented chat
curl http://localhost:20180/v1/web/search \
  -H "Authorization: Bearer YOUR_MASTER_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o", "query": "latest AI news"}'
```

> 🇮🇩 Drop-in replacement untuk OpenAI API — ganti base URL, semua SDK (`openai-python`, `langchain`, `llama-index`, dll) langsung kompatibel.
>
> 🇬🇧 Drop-in replacement for OpenAI API — change the base URL and all SDKs work immediately.

---

## 📊 Dashboard

<details open>
<summary>🇮🇩 17 Halaman Dashboard</summary>

Lintasan dilengkapi dashboard interaktif berbasis **SvelteKit 5 + Tailwind v4** untuk monitoring dan konfigurasi real-time.

| Halaman | Fungsi |
|---------|--------|
| **Overview** | Statistik global — requests, tokens, cache hit rate, latency |
| **Accounts** | Kelola koneksi provider (add/test/sync/delete) |
| **Routing** | Konfigurasi combo + load balancer + aliases |
| **Fallback** | Multi-level fallback chain per model/connection |
| **Logs** | Real-time request log dengan filter & search |
| **Usage** | Token usage + cost per provider/model |
| **Observability** | Exportable `/metrics` + Real-time Grafana-like panels |
| **Analytics** | Metrics dashboard — latency, throughput, savings |
| **API Keys** | Generate, copy, revoke API keys |
| **Teams** | Team-based access control |
| **Users** | User management + role assignment |
| **Webhooks** | Event-driven webhook setup & testing |
| **Backup** | Database backup, restore, export |
| **Settings** | Global configuration — port, keys, limits |
| **Plugins** | Plugin store + management + AI generator |
| **Playground** | Interactive chat console untuk testing API |
| **Models** | Model catalog dengan pricing |
| **Docs** | Built-in API documentation |

Akses: `http://localhost:20180/dashboard` (via nginx reverse-proxy ke SvelteKit di port 5173)

</details>

<details>
<summary>🇬🇧 17 Dashboard Pages</summary>

Lintasan comes with an interactive dashboard built with **SvelteKit 5 + Tailwind v4** for real-time monitoring and configuration.

| Page | Function |
|------|----------|
| **Overview** | Global stats — requests, tokens, cache hit rate, latency |
| **Accounts** | Manage provider connections (add/test/sync/delete) |
| **Routing** | Smart-route configs, combo + load balancer + aliases |
| **Fallback** | Multi-level fallback chains per model/connection |
| **Logs** | Real-time request log with filter & search |
| **Usage** | Token usage + cost per provider/model |
| **Observability** | Exportable `/metrics` + Real-time Grafana-like panels |
| **Analytics** | Metrics dashboard — latency, throughput, savings |
| **API Keys** | Generate, copy, revoke API keys |
| **Teams** | Team-based access control |
| **Users** | User management + role assignment |
| **Webhooks** | Event-driven webhook setup & testing |
| **Backup** | Database backup, restore, export |
| **Settings** | Global configuration — port, keys, limits |
| **Plugins** | Plugin store + management + AI generator |
| **Playground** | Interactive chat console for API testing |
| **Models** | Model catalog with pricing |
| **Docs** | Built-in API documentation |

Access: `http://localhost:20180/dashboard` (via nginx reverse-proxy to SvelteKit at port 5173)

</details>

---

## 🔌 Provider Presets (118 Ready)

<details open>
<summary>🇮🇩 Daftar Provider</summary>

### Major Providers (4)
OpenAI · Anthropic · DeepSeek · Google Gemini

### Top-Tier (6)
xAI (Grok) · Mistral AI · Azure OpenAI · Azure AI Foundry · Google Vertex AI · AWS Bedrock

### AI Coding (4)
Codestral API · GitHub Copilot API · Pydantic AI Agents · Meta Llama API

### Aggregators (8)
OpenRouter · Replicate · HuggingFace Inference · Vercel AI Gateway · AIML API · Poe by Quora · CometAPI · NanoGPT

### High-Speed Inference (11)
Groq · Together AI · Fireworks AI · Cerebras · NVIDIA NIM · Cloudflare Workers AI · Hyperbolic · Lambda AI · FriendliAI · Anyscale Endpoints · SambaNova

### GPU Cloud (12)
Baseten · OctoAI · Lepton AI · Featherless AI · Crusoe Cloud · nscale AI · PublicAI · Galadriel · Chutes · GMI Cloud · Heroku AI · Novita AI

### CommandCode (2)
CommandCode (API Key) · CommandCode (Alpha)

### Chinese Providers (10)
GLM / Zhipu AI · Kimi / Moonshot · MiniMax · Qwen / Alibaba · SiliconFlow · Xiaomi MiMo · Volcano Engine (ByteDance) · Z.AI · DeepSeek · Baidu Qianfan

### Indonesia Providers (2)
Sumopod · Apertis AI (Stima API)

### Enterprise & Cloud (8)
Snowflake Cortex AI · Oracle Cloud (OCI) · SAP AI Core · IBM watsonx · Gradient AI · NLP Cloud · Petals · Clarifai

### Specialized (19)
Perplexity · Cohere · DeepInfra · Nebius AI · Aleph Alpha · AI21 Labs · Reka AI · Voyage AI · Deepgram · Black Forest Labs · Stability AI · Runway ML · Recraft AI · fal.ai · Helicone · Lemonade AI · Bytez · Sarvam AI · MorphDB

### Self-Hosted (3)
Ollama · vLLM · LM Studio

### Other (24)
AWS SageMaker · GigaChat · Predibase · OpenPipe · Scale AI · Titan ML · OctoML · Monster API · GooseAI · Forefront · Custom

</details>

---

## 📂 Struktur Project / Project Structure

```
lintasan/
├── cmd/lintasan/              # CLI entry point (Cobra)
├── internal/
│   ├── auth/                  # OAuth + API key auth
│   ├── config/                # Environment & DB config
│   ├── dashboard/             # (deprecated, moved to frontend/)
│   ├── db/                    # SQLite database layer
│   ├── discover/              # Model auto-discovery
│   ├── freeproviders/         # Free provider scanner (118+ presets)
│   ├── expprovider/           # AES-256-GCM Credentials
│   ├── memory/                # Vector memory (pluggable embedder)
│   ├── mitm/                  # MITM HTTPS bridge
│   ├── mlrouter/              # ML-based routing
│   ├── plugin/                # JS plugin engine
│   ├── rtk/                   # RTK token compressor
│   ├── server/                # HTTP server + all handlers
│   │   ├── server.go          # Core server + middleware
│   │   ├── proxy.go           # Provider proxy + streaming
│   │   ├── router.go          # Smart routing logic
│   │   ├── cache.go           # Settings cache
│   │   ├── handlers_*.go      # API handlers (dashboard, parity, etc.)
│   │   └── middleware_*.go    # CORS, auth middleware
│   └── websearch/             # Web search engine
├── frontend/                  # SvelteKit dashboard (Go+Svelte stack)
│   ├── src/
│   │   ├── routes/            # 17 page routes
│   │   ├── lib/               # Components, stores, utils
│   │   └── app.html           # Root HTML template
│   ├── static/                # Static assets
│   ├── svelte.config.js       # SvelteKit config (adapter-node)
│   ├── vite.config.ts         # Vite config
│   └── package.json           # Frontend dependencies
├── data/                      # Runtime data DB (auto-created)
├── docs/                      # Documentation
│   ├── api-reference.md       # Full API reference (deprecated, read AGENTS.md)
│   └── design-system.md       # UI design system
├── .env.example               # Example environment file
├── Dockerfile                 # Docker build
├── docker-compose.yml         # Docker compose
├── go.mod / go.sum            # Go dependencies
├── Makefile                   # Build automation
├── LICENSE                    # MIT
└── README.md                  # This file
```

---

## 💻 Development

```bash
# Clone
git clone https://github.com/sanhaji182/lintasan.git
cd lintasan

# Backend
go run ./cmd/lintasan start        # Run server (hot-reload with air if installed)

# Frontend
cd frontend
npm run dev -- --port 5173        # Dev UI + Vite server with HMR (Requires GO RUNNING)

# Build all
make build                         # go build + npm run build

# Run tests
go test ./...                      # 700+ tests
cd frontend && npm run check       # SvelteKit type-check
```

---

## 🧪 Testing

```bash
# All backend tests
go test ./...                      # 700+ passed, 0 failed, 39 packages

# With coverage
go test -cover ./...

# Specific package
go test -v ./internal/server/...

# Frontend
cd frontend && npm run check       # SvelteKit type-check + lint
```

---

## 🚢 Deployment

<details open>
<summary>🇮🇩 Production Deployment</summary>

```bash
# Build statically linked production binary
cd lintasan
go build -ldflags="-s -w" -o lintasan ./cmd/lintasan

# Build frontend inside go executable
cd frontend && npm install && npm run build && cd ..

# Copy to server
scp lintasan user@server:/opt/lintasan/
scp -r frontend/build user@server:/opt/lintasan/frontend-build/

# Systemd service
sudo tee /etc/systemd/system/lintasan.service << 'EOF'
[Unit]
Description=Lintasan LLM Proxy Gateway
Documentation=https://github.com/sanhaji182/lintasan
After=network.target

[Service]
Type=simple
User=lintasan
WorkingDirectory=/opt/lintasan
ExecStart=/opt/lintasan/lintasan start
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable --now lintasan
```

### Nginx Reverse Proxy

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;

    # API → Go backend
    location /api/ { proxy_pass http://127.0.0.1:20180; }
    location /v1/  { proxy_pass http://127.0.0.1:20180; }

```

</details>

---

## 🏆 Benchmark

Duel head-to-head melawan 9Router — backend sama (CommandCode + deepseek-v4-pro).

*Head-to-head duel against 9Router — same backend (CommandCode + deepseek-v4-pro).*

| Task | Lintasan | 9Router | Winner |
|------|----------|---------|--------|
| LRU Cache implementation | 11.2s | 11.6s | Lintasan |
| Code review merge_sorted | 7.2s | 10.9s | Lintasan |
| TypeScript deep-merge generic | 14.7s | 15.4s | Lintasan |
| Optimistic vs pessimistic locking | 11.1s | 15.3s | Lintasan |
| Docker exits code 0 | 12.9s | 6.3s | 9Router |
| GitHub Actions workflow | 8.0s | 3.7s | 9Router |
| Rate limiting middleware | 6.0s | 4.1s | 9Router |
| REST vs GraphQL vs gRPC | 11.5s | 15.4s | Lintasan |

**Results:** Cold avg 10.3s vs 10.3s (parity) · **Lintasan 5 — 9Router 3** · Thinking leaked: Lintasan 1/8 vs 9Router 3/8

---

## 🔄 Migration dari Node.js v1

<details open>
<summary>🇮🇩 Panduan Migrasi</summary>

Lintasan Go menggunakan database schema yang **berbeda** dari Node.js v1 — tidak backward-compatible.

1. **Export** settings dari Node v1 dashboard
2. **Install** Lintasan Go (lihat [Instalasi](#-instalasi--installation))
3. **Re-create** connections di Go v2 dashboard atau via API
4. **Sync** models — auto-discovered saat pertama connect
5. **Verifikasi** — test lewat Playground

> ⚠️ Repo Node.js v1 sudah di-archive di [sanhaji182/lintasan](https://github.com/sanhaji182/lintasan)

</details>

<details>
<summary>🇬🇧 Migration Guide</summary>

Lintasan Go uses a **different** database schema from Node.js v1 — not backward-compatible.

1. **Export** settings from Node v1 dashboard
2. **Install** Lintasan Go (see [Installation](#-installation))
3. **Re-create** connections in Go v2 dashboard or via API
4. **Sync** models — auto-discovered on first connection
5. **Verify** — test via Playground

> ⚠️ Node.js v1 repo is archived at [sanhaji182/lintasan](https://github.com/sanhaji182/lintasan)

</details>

---

## 🤝 Contributing

<details open>
<summary>🇮🇩 Kontribusi</summary>

Kontribusi sangat diterima! Buka issue untuk bug report atau feature request.

1. Fork repo
2. Buat branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'feat: amazing feature'`)
4. Push ke branch (`git push origin feature/amazing-feature`)
5. Buka Pull Request

Pastikan tests tetap passing:
```bash
go test ./...     # Harus 0 failures
```

</details>

<details>
<summary>🇬🇧 Contributing</summary>

Contributions welcome! Open an issue for bug reports or feature requests.

1. Fork the repo
2. Create a branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'feat: amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Make sure tests still pass:
```bash
go test ./...     # Must be 0 failures
```

</details>

---

## 📄 Lisensi / License

MIT

---

## 🤖 Built with AI

<details open>
<summary>🇮🇩</summary>

Project ini dibangun dengan AI sebagai development partner. Arsitektur, keputusan teknis, dan quality control tetap di tangan manusia — AI mengeksekusi.

Lintasan Go adalah rewrite total dari Node.js v1 — mempertahankan semua fitur dengan footprint 35x lebih ringan. Dibuat untuk komunitas AI Indonesia yang percaya bahwa masa depan development adalah orkestrasi, bukan sekadar coding manual.

</details>

<details>
<summary>🇬🇧</summary>

This project was built with AI as a development partner. Architecture, technical decisions, and quality control remain in human hands — AI executes.

Lintasan Go is a complete rewrite of Node.js v1 — retaining all features with 35x lighter footprint. Made for the AI community that believes the future of development is orchestration, not just manual coding.

</details>

**Orchestrator:** [Sanhaji](https://github.com/sanhaji182) · Programmer · AI-assisted development advocate

---

<p align="center">
  <b>🇮🇩 Lintasan Go (v2)</b> — Setiap Koneksi Punya Jalannya<br>
  <b>🇬🇧 Lintasan Go (v2)</b> — Every Connection Has Its Path<br><br>
  Dibangun dengan 🤖 AI · Diorkestrasi oleh 👨‍💻 <a href="https://github.com/sanhaji182">Sanhaji</a>
</p>
