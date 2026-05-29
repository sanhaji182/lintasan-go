# AGENTS.md — Lintasan Go

> **Baca file ini dulu sebelum menyentuh kode apa pun.** Ini adalah peta sistem untuk AI agent (Claude Code, Codex, Cursor, Hermes, dll). Tujuannya: paham keseluruhan arsitektur, konvensi, dan cara kerja Lintasan tanpa harus meraba-raba.

---

## 1. Apa itu Lintasan

Lintasan adalah **LLM proxy gateway** — satu endpoint OpenAI-compatible yang merutekan request ke banyak provider AI (OpenAI, Anthropic, DeepSeek, Gemini, Groq, dll) dengan smart routing, failover, caching, token compression, observability, dan dashboard.

- **Backend:** Go (single binary, ~24MB)
- **Frontend:** SvelteKit 5 (dashboard, 26+ halaman)
- **Repo:** `github.com/sanhaji182/lintasan-go` (monorepo)
- **Filosofi:** "Setiap Koneksi Punya Jalannya."

> ⚠️ **Node.js v1 sudah DIHAPUS (May 2026).** Semua kode aktif ada di monorepo ini. Jangan cari/buat referensi ke `~/lintasan` lama.

---

## 2. Arsitektur Tingkat Tinggi

```
                 ┌─────────────────────────────────────────┐
   Browser  ───► │  nginx (lintasan.sans.biz.id, :443 TLS)  │
                 └───────────────┬─────────────┬────────────┘
                                 │             │
              /api/* /v1/* /health│             │/ (UI lainnya)
                                 ▼             ▼
                   ┌──────────────────┐  ┌──────────────────┐
                   │  Go backend      │  │  SvelteKit (SSR)  │
                   │  :20180          │  │  :5173 (node)     │
                   │  lintasan start  │  │  build/index.js   │
                   └────────┬─────────┘  └──────────────────┘
                            │
                            ▼
            ┌───────────────────────────────────┐
            │ SQLite (data) + provider upstreams │
            └───────────────────────────────────┘
```

**Pembagian routing nginx (penting):**
- `/api/*` → Go `:20180` (dashboard API)
- `/v1/*` → Go `:20180` (OpenAI-compatible LLM proxy)
- `/health` → Go `:20180`
- `/` dan sisanya → SvelteKit `:5173` (UI)

Frontend memanggil API lewat path relatif (`/api/...`, `/v1/...`), jadi nginx yang memisahkan ke backend. **Jangan hardcode `localhost:20180` di frontend.**

---

## 3. Port & Service Map

| Service | Port | systemd unit | WorkingDir | ExecStart |
|---------|------|--------------|------------|-----------|
| Go backend | `20180` | `lintasan.service` | `/home/ubuntu/lintasan-go` | `lintasan start` |
| SvelteKit dashboard | `5173` | `lintasan-dashboard.service` | `/home/ubuntu/lintasan-go/frontend` | `node build/index.js` |

Env penting dashboard: `PORT=5173`, `HOST=0.0.0.0`, `ORIGIN=https://lintasan.sans.biz.id`.
Env penting backend: `PORT=20180`.

**Kedua service jalan sebagai systemd (Restart=always, PPID=1).** Jangan jalankan sebagai child process sesi — akan mati saat sesi putus.

Restart setelah deploy:
```bash
sudo systemctl restart lintasan-dashboard lintasan
sudo systemctl is-active lintasan-dashboard lintasan
```

---

## 4. Struktur Repo

```
lintasan-go/
├── cmd/                      # entrypoint binary (lintasan start)
├── internal/                 # 36 package backend (lihat §5)
├── frontend/                 # SvelteKit 5 dashboard
│   ├── src/
│   │   ├── routes/           # halaman (file-based routing)
│   │   │   ├── +page.svelte           # landing page (/)
│   │   │   ├── login/+page.svelte      # login
│   │   │   └── dashboard/              # semua halaman dashboard
│   │   │       ├── +layout.svelte      # shell (Sidebar + Header)
│   │   │       ├── +layout.ts          # AUTH GUARD (client-side)
│   │   │       └── <page>/+page.svelte
│   │   ├── lib/
│   │   │   ├── api.ts                  # API client (auto-attach JWT)
│   │   │   ├── components/             # Sidebar, Header, Toast, dll
│   │   │   └── stores/                 # theme, auth, toast
│   │   └── app.css                     # design tokens (CSS vars)
│   └── build/                # output produksi (node adapter)
├── docs/
│   ├── api-reference.md
│   └── design-system.md
├── README.md                 # dokumentasi lengkap (bilingual)
├── CONTRIBUTING.md
└── AGENTS.md                 # file ini
```

---

## 5. Backend Packages (`internal/`)

Setiap package punya tanggung jawab tunggal. Saat menambah fitur, ikuti pola package yang sudah ada.

| Package | Fungsi |
|---------|--------|
| `server` | HTTP mux, route registration, middleware (18 file, ~70 route) |
| `proxy` | Core LLM proxy: chat completions, embeddings, images, audio |
| `auth` | JWT auth (HS256), password hashing (SHA-512), user CRUD, middleware |
| `config` | Loading & validasi konfigurasi |
| `db` | SQLite schema + migrations |
| `cache` | Semantic cache (cosine similarity), response/stream cache |
| `combo` | Routing combos (alias model → provider+model) |
| `fallback` | Fallback chain antar provider/model |
| `circuit` | Circuit breaker (auto-disable provider gagal) |
| `compress` | Token compression |
| `cost` | Cost tracking per request |
| `budget` | Budget limits |
| `discover` | Provider discovery |
| `freeproviders` | Free provider catalog |
| `guard` | Guardrails (input/output filtering) |
| `lb` | Load balancing |
| `mcp` | MCP server (JSON-RPC 2.0, 14 tools) — HTTP + SSE |
| `memory` | Vector memory (search/store/stats/list) |
| `mlrouter` | ML-based smart routing |
| `models` | Model catalog & metadata |
| `optimizer` | Request optimization |
| `plugin` | Plugin system (extensible tanpa ubah core) |
| `quality` | Quality scoring |
| `quota` | Quota management |
| `ratelimit` | Rate limiting |
| `reasoning` | Reasoning extraction |
| `reflect` | Self-reflection |
| `retry` | Retry logic |
| `translator` | Cross-format translation (5 format) |
| `webhook` | Webhook subscriptions |
| `websearch` | Web search integration |
| `logging` | Request logging |
| `batch`, `mitm`, `rtk` | Batch processing, MITM inspect, runtime toolkit |

---

## 6. API Route Map

Auth (JWT dashboard):
```
POST   /api/auth/login        # username+password → { token, user }
GET    /api/auth/me           # validasi token (Bearer)
POST   /api/auth/logout
GET    /api/auth/users        # admin only
POST   /api/auth/users        # admin only
```

OpenAI-compatible proxy (`/v1/*`):
```
GET    /v1/models
POST   /v1/chat/completions   # streaming + non-streaming
POST   /v1/embeddings
POST   /v1/images/generations
POST   /v1/audio/speech
POST   /v1/audio/transcriptions
POST   /v1/web/search
GET    /v1/memory  ·  GET /v1/memory/search  ·  POST /v1/memory  ·  GET /v1/memory/stats  ·  DELETE /v1/memory/{key}
```

Dashboard API (`/api/*`):
```
GET    /api/models/catalog
GET    /api/connections · POST · PATCH · DELETE (+ /{id})
GET    /api/combos · POST · PUT · DELETE
GET    /api/providers/discover
GET    /api/stats · /api/logs · /api/analytics · /api/telemetry
GET    /api/savings/summary · /api/savings/history
POST   /api/translate · GET /api/translate/formats
GET    /api/mcp/tools
```

MCP (JSON-RPC 2.0):
```
POST   /mcp        # HTTP transport
GET    /mcp/sse    # Server-Sent Events transport
```

Health:
```
GET    /health
```

> Daftar lengkap & contoh payload ada di `docs/api-reference.md`.

---

## 7. Auth Flow (WAJIB dipahami sebelum sentuh dashboard)

Kontrak UX auth (jangan dirusak):

1. **Unauthenticated → selalu ke `/login`.**
   - `dashboard/+layout.ts` adalah guard client-side: kalau tidak ada token → `redirect(307, '/login')`; kalau ada → validasi `GET /api/auth/me` dengan Bearer token; gagal → clear localStorage + redirect login.
   - `/` (`+page.svelte`) tidak hardcode ke dashboard — route by auth state.

2. **Token ada ≠ session valid.** Selalu validasi via `/api/auth/me` sebelum auto-forward dari login ke dashboard.

3. **Kontrol auth eksplisit di shell:**
   - Header: tombol `Login` (belum auth) / `Logout` + shortcut user (sudah auth).
   - Sidebar: label `User Management` untuk admin akun.

Token disimpan di `localStorage`: `lintasan_token`, `lintasan_user`.
API client (`frontend/src/lib/api.ts`) otomatis attach `Authorization: Bearer <token>` dan handle 401 (clear + redirect login).

**Default kredensial:** `admin` / `admin123` (seed admin di `internal/auth/migration.go`).

---

## 8. Frontend Konvensi

- **SvelteKit 5** dengan runes (`$state`, `$derived`, `$props`, `$bindable`).
- **Design tokens** di `frontend/src/app.css` sebagai CSS variables (`--color-*`, `--radius-*`, `--shadow-*`). Default theme = **clean light**. Dark mode tersedia lewat `[data-theme="dark"]` + `theme` store.
- **Styling:** Tailwind v4 + scoped `<style>` per komponen. Untuk halaman entry (landing/login) pakai scoped CSS standalone agar konsisten.
- **Ikon:** `lucide-svelte`.
- **API calls:** selalu lewat `api` dari `$lib/api` (jangan `fetch` mentah kecuali untuk auth/me di guard).
- **A11y wajib:** `<label for>` + `id`, tombol icon-only butuh `aria-label`/`title`, modal butuh role + keyboard handler, hindari mouseenter/mouseleave JS untuk hover (pakai CSS `:hover`). Build harus bebas warning a11y.

Pitfall lama (sudah resolved tapi catat): lucide-svelte sebagai komponen di `EmptyState icon={...}` kadang memicu type error svelte-check; ini warning type, bukan blocker build.

---

## 9. Build, Test, Deploy

**Backend:**
```bash
cd /home/ubuntu/lintasan-go
go build -o lintasan ./cmd/...     # build binary
go test ./...                       # 508 tests, 35 packages — harus PASS
```

**Frontend:**
```bash
cd /home/ubuntu/lintasan-go/frontend
npm run build                       # harus PASS, target zero a11y warning
npm run check                       # svelte-check (type/a11y diagnostics)
```

**Deploy (setelah build):**
```bash
sudo systemctl restart lintasan-dashboard lintasan
sudo systemctl is-active lintasan-dashboard lintasan
```

**Verifikasi live:**
- `https://lintasan.sans.biz.id/` → landing
- `/dashboard` tanpa token → redirect `/login`
- login `admin/admin123` → dashboard
- logout → `/login`

---

## 10. Aturan untuk Agent

1. **Jangan rusak auth flow** (§7). Setiap perubahan dashboard harus tetap lulus: unauth→login, validasi `/api/auth/me`, login→dashboard, logout→login.
2. **Jangan ubah perilaku API/endpoint** kecuali memang itu task-nya. Frontend & klien eksternal bergantung pada kontrak ini.
3. **Build harus tetap hijau.** Jalankan `go test ./...` dan `npm run build` sebelum klaim selesai.
4. **Commit per task**, pesan jelas (`feat(...)`, `fix(...)`). Push ke `main` hanya setelah build pass.
5. **Pure Go, minim dependency** — stdlib + `go-sqlite3` adalah preferensi. Jangan tambah dependency berat tanpa alasan kuat.
6. **Default theme clean light** — hindari desain dark-heavy/glow berlebihan untuk halaman entry.
7. **Verifikasi nyata** — pakai browser/curl untuk cek hasil sebelum klaim "done", jangan berasumsi.
8. **Secrets** — jangan commit `.env`, API key, atau credential. Pakai `.env.example` untuk dokumentasi env.

---

## 11. Quick Reference

```bash
# Status service
systemctl status lintasan lintasan-dashboard --no-pager

# Health check
curl -s https://lintasan.sans.biz.id/health

# Login (dapat token)
curl -s -X POST https://lintasan.sans.biz.id/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'

# Chat completion (OpenAI-compatible)
curl -s -X POST https://lintasan.sans.biz.id/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"<combo-or-model>","messages":[{"role":"user","content":"hi"}]}'

# Logs service
journalctl -u lintasan -n 50 --no-pager
journalctl -u lintasan-dashboard -n 50 --no-pager
```

---

*Last updated: 2026-05-29 · Stack: Go 1.22.2 + SvelteKit 5 · 508 backend tests · 36 internal packages · 70 routes*
