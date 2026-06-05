# AGENT.md — Lintasan Single Source of Truth

> **Tujuan file ini:** memberi agent baru (Claude Code, Codex, Cursor, Hermes, dll) gambaran **state proyek saat ini** tanpa harus membaca ulang roadmap, proposal, thread engineering, atau chat history. Ini **bukan arsip sejarah** — ini *current truth*. Untuk peta arsitektur teknis (port, package, route, deploy), baca `AGENTS.md`.
>
> **Aturan konflik:** kalau dokumen lama berbeda dengan state repo/runtime sekarang, **repo + runtime menang**. Semua angka di bawah diverifikasi langsung dari `git` dan dari mesin prod, bukan dari ingatan.
>
> **Terakhir direkonsiliasi:** 2026-06-05, terhadap `main @ 17322f6` dan prod binary `7b273520` (`v0.24.0-1-g17322f6`).

---

## 1. Executive Summary

Lintasan adalah **LLM proxy gateway** Go (single binary) + dashboard SvelteKit, OpenAI-compatible, dengan smart routing, failover, caching, observability. Repo: `github.com/sanhaji182/lintasan-go` (monorepo).

Dua jalur kerja besar yang aktif sekarang:

1. **Official Layer (Provider SDK + Capability System)** — sudah masuk prod. F1 (Provider SDK) + F2.0–F2.5 (capability vocabulary, catalog, resolver, shadow, enforcement, embedder path) semuanya **merged ke main** dan **terkompilasi di binary prod**. F2.4 capability enforcement **armed (flag ON)** di prod.

2. **Experimental Ecosystem (ACP Shape-2 agents)** — foundation + **Generic ACP Provider Framework** + **Codex onboarding** + **Codex wire-remediation** + **Cohort-A readiness** semuanya **merged ke main DAN terkompilasi di binary prod** (sejak deploy 2026-06-05). Framework generik sudah jadi **baseline**: onboarding provider = descriptor + credential mapping + fixture, bukan kode flow baru. Cohort-A lainnya (Claude Code, Gemini CLI, Copilot) punya descriptor + mapping + fixture skeleton (**readiness**, belum live-validated). Semua **additive + dormant + membrane-gated**, sudah di prod binary tapi **belum diaktifkan**.

**Posisi singkat:** Official Layer = produksi. Experimental Layer = **sudah masuk prod binary, tidur** (membrane-gated, tak ada auto-activation). Framework generik baseline; Codex nunggu kredensial untuk validasi M5 akhir; Cohort-A lain nunggu live validation masing-masing.

---

## 2. Current Production State

- **Service:** `lintasan.service` (systemd, `Restart=always`), port `20180`, single binary serving UI + API. **Status: active.**
- **Running binary:** SHA256 `7b273520…`, versi `v0.24.0-1-g17322f6` → **dibangun di commit `17322f6`**.
- **Binary drift:** TIDAK ADA. SHA `/proc/<pid>/exe` == SHA `./lintasan` on-disk. Running == deployed.
- **Prod binary = main** (`main @ 17322f6`, `HEAD` = `main` = `origin/main`, no drift). Versi: `v0.24.0` (tag) + 1 commit (the curl-import hot-fix).
- **Versi scheme (2026-06-05 reset):** Lintasan sekarang 0.x.x (pre-1.0), bukan 2.x.x. Tag `v2.4.0` tetap di history sebagai referensi. Lihat CHANGELOG untuk rationale.
- **Flag prod (DB `settings`, `data/lintasan.db`):**
  - `capability_enforce_enabled = true` (F2.4 **armed**)
  - `capability_shadow_enabled = true` (F2.3 shadow ON)
- **Yang AKTIF di hot path prod:** capability system (F2.0–F2.5), F2.4 enforcement, membrane (struktural), curl import endpoint (`POST /api/connections/import-curl`). Provider SDK (F1) flag-gated.
- **Yang DORMANT di binary prod:** Experimental substrate G1–G6 (`internal/expprovider`), Generic ACP Provider Framework, Codex onboarding + wire-remediation, Cohort-A descriptors (Claude/Gemini/Copilot). Semua additive+dormant; membrane satu-arah mencegah auto-routing. Aktivasi = checkpoint operator-gated terpisah.

> ⚠️ Prod DB aktual = `/home/ubuntu/lintasan-go/data/lintasan.db`. (`./lintasan.db` dan `~/.lintasan/lintasan.db` kosong — jangan tertipu.)

---

## 3. Foundation Status

Semua merged ke `main`. Yang berbintang (★) sudah di binary prod.

| Layer | Status | Di prod binary? |
|-------|--------|-----------------|
| F1 — Provider SDK (Official Layer wiring) | ✅ merged, CLOSED | ★ ya (flag-gated) |
| F2.0 — Canonical Capability Vocabulary | ✅ merged | ★ ya |
| F2.1 — Capability Lookup Integration (read-only) | ✅ merged | ★ ya |
| F2.2 — Capability Catalog + `/api/capabilities` | ✅ merged | ★ ya |
| F2.3 — Shadow routing + 3-tier resolver (observe-only) | ✅ merged | ★ ya |
| F2.4 — Capability eligibility enforcement (flag-gated) | ✅ merged | ★ ya, **flag ON** |
| F2.5 — Embedder execution path (flag-gated, byte-parity) | ✅ merged | ★ ya |
| F2-membrane — Official/Experimental one-way routing | ✅ merged | ★ ya |
| Phase 3 — E1 process isolation harness | ✅ merged | ★ ya (dormant) |
| Phase 4 — ACP integration layer (JSON-RPC broker) | ✅ merged | ★ ya (dormant) |
| Experimental substrate G1–G6 (`internal/expprovider`) | ✅ merged | ★ ya (dormant) |
| ACP wire reconciliation (spec-faithful broker) | ✅ merged | ★ ya (dormant) |
| Codex onboarding (provider #1) | ✅ merged | ★ ya (dormant) |
| Codex wire-remediation (authenticate/sessionId/ContentBlock/cwd) | ✅ merged | ★ ya (dormant) |
| Generic ACP Provider Framework | ✅ merged, **BASELINE** | ★ ya (dormant) |
| Cohort-A readiness (Claude/Gemini/Copilot descriptors) | ✅ merged | ★ ya (dormant) |
| Curl import endpoint (`/api/connections/import-curl`) | ✅ merged (`17322f6`) | ★ ya, **ACTIVE** |

---

## 4. Experimental Ecosystem Taxonomy

**Prinsip inti:** Official Layer dan Experimental Layer dipisah oleh **membrane satu-arah**. Production routing (`ResolveRoutable`) **tidak akan pernah** memilih provider Experimental. Experimental hanya dijangkau lewat **pintu eksplisit** (`ResolveExperimental`) + **sinyal opt-in** (model prefix `experimental/<name>`, atau header `X-Lintasan-Track: experimental` + `X-Lintasan-Provider: <name>`). Tanpa sinyal eksplisit, request tidak mungkin mendarat di Experimental.

**Substrate G1–G6** (`internal/expprovider`, provider-agnostic, additive+dormant):
- **G1** adapter seam — `ACPProvider` (provider.Provider + Agent exec interface)
- **G2** launcher registry — `LaunchSpec` (cara meluncurkan agent)
- **G3** routing entry — `DetectExperimental` (parser sinyal opt-in)
- **G4** credential injection — `Injector` (secret per-provider, scoped, Invariant 3)
- **G5** admission harness — gates Isolation + Protocol + Acceptance + Membrane
- **G6** lifecycle/badge — state machine + RiskBadge

**Generic ACP Provider Framework** (`framework.go`, baseline sejak `52647d1`) — engine provider-agnostic:
- `ProviderDescriptor` — definisi deklaratif satu provider (name, executable, args, AuthMode, AuthEnvVar, AuthMethodID, capabilities, foreign-secret set, cwd, timeouts). **Satu-satunya input per-provider.**
- `AdmitProvider(ctx, reg, spec, caps, src, foreignAuthVars)` — admission flow generik (register → proposed→admitted → harness → active-on-GO), membrane invariant dibuktikan per-provider. **Tidak ada nama provider hardcoded.**
- `ExperimentalHarness`, `NewExperimentalProvider`, dan 3 probe (`IsolationProbe`/`ProtocolProbe`/`AcceptanceProbe`) + `grantingPermission` — semua provider-agnostic, dipakai ulang verbatim.
- **Aturan onboarding (operator-locked):** provider baru = **descriptor + credential mapping + fixture**. DILARANG mengubah framework untuk menambah provider.

**Track taxonomy:**
- **Official** — provider resmi, routable, capability-trusted untuk routing.
- **Experimental** — `Track()==Experimental`, NEVER routable, capability declared-only (Invariant 5: tak pernah dipercaya untuk routing).

**ACP Shape membedakan dua arah orthogonal:**
- **Shape 1** — agent menunjuk **ke** Lintasan sebagai model endpoint (mis. Codex Responses ingress `/v1/responses`). Branch `feat/codex-m0-skeleton`, terpisah.
- **Shape 2** — Lintasan menggerakkan CLI agent resmi sebagai subprocess via ACP over stdio (mis. `codex-acp`). Inilah jalur Experimental provider di doc ini.

---

## 5. ACP Status

- **Broker** (`internal/experimental/acp.go`): **spec-faithful** (agentclientprotocol.com), bukan dialek satu agent. Satu broker melayani semua agent ACP (Codex, nanti Claude Code/Gemini/Copilot).
- **Fakta wire yang terkunci (diverifikasi lawan codex-acp 0.15.0 asli):**
  - `protocolVersion` = **integer** `1` (bukan string).
  - Lifecycle: `initialize` → `authenticate` (kalau agent minta) → `session/new {cwd, mcpServers}` → `session/prompt {sessionId, prompt:[ContentBlock]}` → stream → terminal `stopReason`.
  - Prompt turn = STREAM: 0..n `session/update` notification (tak dibalas) + 0..n `session/request_permission` request (wajib dibalas) → terminal response.
  - Tools **di-report + dieksekusi oleh agent**; host cuma consent (`session/request_permission`). Host TIDAK menjalankan tool.
  - toolCallId dibawa **verbatim** (identifier fidelity).
  - Client capabilities (fs/*, terminal/*) **tidak diiklankan** → agent spec-forbidden memanggilnya; broker balas method-not-found (defense-in-depth).
- **Status:** merged ke main (`0a6bbd6`) + wire remediation Codex (`4794542`) sudah ikut ke main. Broker = onboarding-ready untuk semua Cohort-A.

---

## 6. Codex Status

Codex = **Experimental Provider #1 (Cohort A)**. File: `internal/expprovider/codex.go` (sekarang thin descriptor di atas framework).

- **Onboarding implementation: COMPLETE** (merged ke `main`).
- **Wire validation: PASS** (diverifikasi lawan binary codex-acp 0.15.0 asli, sampai batas model-call).
- **Status lifecycle:** implemented + admitted-capable + **membrane-gated** + **NOT production-validated**.
- **Apa yang ada:** `CodexDescriptor` (path `codex-acp`, auth `OPENAI_API_KEY`, methodId `openai-api-key`) + wrapper backward-compat (`CodexLaunchSpec`/`NewCodexProvider`/`CodexHarness`/`AdmitCodex` delegate ke framework).
- **codex-acp binary:** terinstall di `/tmp/codex-acp-install/node_modules/@zed-industries/codex-acp-linux-x64/bin/codex-acp` (real ELF 0.15.0). **Bukan** terinstall global/permanen — env scratch untuk validasi.
- **Orthogonal:** Codex Shape-1 Responses ingress (`/v1/responses`, branch `feat/codex-m0-skeleton`) **terpisah** dari Shape-2 provider ini. Jangan campur.

---

## 6b. Cohort-A Readiness (Claude Code / Gemini CLI / Copilot)

Ketiga provider sisa Cohort-A punya **descriptor + credential mapping + fixture skeleton** (readiness penuh), dan terbukti **onboardable via framework generik tanpa perubahan engine** (`TestCohortA_AllProvidersOnboard_NoFrameworkChange`).

| Provider | Descriptor | Auth env var | ACP method id (UNVERIFIED) | Fixture skeleton | Live-validated? |
|----------|-----------|--------------|----------------------------|------------------|-----------------|
| claude-code | `ClaudeCodeDescriptor()` | `ANTHROPIC_API_KEY` | `anthropic-api-key` | `testdata/claude-code-session.jsonl` | ❌ belum |
| gemini-cli | `GeminiCLIDescriptor()` | `GEMINI_API_KEY` | `gemini-api-key` | `testdata/gemini-cli-session.jsonl` | ❌ belum |
| copilot | `CopilotDescriptor()` | `GITHUB_TOKEN` | `github-token` | `testdata/copilot-session.jsonl` | ❌ belum |

- **Credential mapping:** `CohortACredentialSource()` mendaftarkan binding provider→env-var (scoped, Invariant 3) via `EnvCredentialSource` — interface yang sama dengan Codex, nol mesin kredensial baru.
- ⚠️ **WIRE-UNVERIFIED:** executable name, auth method id, args, dan auth model (Copilot mungkin device-auth, bukan plain token) adalah **hipotesis**. Pelajaran Codex berlaku: wire contract asli WAJIB dikonfirmasi lawan binary asli di live validation masing-masing provider sebelum descriptor dipercaya. Fixture skeleton itu spec-shaped, **bukan** rekaman CLI asli.
- **Belum:** belum di-register di prod, belum admitted live, belum divalidasi. Readiness = code path siap; live validation = checkpoint operator-gated terpisah per provider.

---

## 7. Validation Status

| Aspek | Status | Bukti |
|-------|--------|-------|
| Codex onboarding implementation | ✅ COMPLETE | merged `7f96a7e` |
| ACP wire contract (vs codex-acp asli) | ✅ PASS | lifecycle initialize→authenticate→session/new[sessionId asli]→session/prompt[accepted] |
| Protocol Gate (in-process) | ✅ PASS | fixture + strict scripted agent |
| Acceptance Gate (in-process, tool-loop close) | ✅ PASS | Codex + Fix4 + framework genericity + Cohort-A tests |
| Generic framework genericity | ✅ PASS | synthetic `acme-agent` onboard, zero framework code |
| Cohort-A onboardability (Claude/Gemini/Copilot) | ✅ PASS (in-process, fixture skeleton) | `TestCohortA_AllProvidersOnboard_NoFrameworkChange` |
| Full repo suite | ✅ pass / 0 fail / 44 pkg / 788 tests | `main @ 17322f6` |
| **Codex Full M5 live (tool-loop close vs OpenAI nyata)** | ⛔ **PENDING** | **environment blocker** (`OPENAI_API_KEY` valid) |
| **Cohort-A lain live (real CLI)** | ⛔ **PENDING** | per-provider operator-gated checkpoint |

**Codex blocker = environment, BUKAN code.** Wire fix terbukti benar sampai batas model-call: dengan dummy key, error bergeser dari `-32602` (wire, di session/new) ke `-32603 / 401 invalid_api_key` (model boundary). Yang kurang: `OPENAI_API_KEY` valid (+ saldo).

---

## 8. Open Branches

`main = origin/main = 17322f6`. Branch yang relevan:

| Branch | State vs main | Catatan |
|--------|---------------|---------|
| `feat/codex-live-remediation` | merged | Sudah FF ke main (`4794542`). Aman dihapus. |
| `feat/acp-generic-framework` | merged | Sudah FF ke main (`52647d1`). Aman dihapus. |
| `feat/cohort-a-readiness` | merged | Sudah FF ke main (`0a3f176`). Aman dihapus. |
| `feat/codex-onboarding`, `feat/acp-wire-reconciliation`, `feat/f1-official-wiring`, `feat/f2.*` | merged | Sudah di main. Aman dihapus. |
| `feat/curl-import-connection` | unmerged | Curl-import feature: 2 commit di depan main (proxy.go, provider_bootstrap.go debug). Lapisan utama `curl_import.go` sudah di-land terpisah via `17322f6`. Tinggal rebase + merge kalau perubahan proxy itu masih relevan. |
| `feat/codex-m0-skeleton` | unmerged | **Shape-1** Codex Responses ingress (multi-agent shim). Orthogonal, lifecycle terpisah. Jangan campur dengan Shape-2. |

Branch lain (`fix/*`, `feat/smart-routing`, `feat/observability`, `feat/compress-bench-harness`, dll) di luar scope dokumen ini.

---

## 9. Pending Decisions

1. **Sediakan `OPENAI_API_KEY` valid** untuk validasi M5 penuh Codex. Gating item utama Codex.
2. ~~**Deploy main ke prod binary?**~~ **RESOLVED (2026-06-05):** prod binary = main @ 17322f6. Experimental Layer sudah masuk binary, dormant (membrane-gated, tak ada auto-activation). Tidak ada langkah deploy terpisah yang perlu.
3. **Live validation Cohort-A** (Claude/Gemini/Copilot) — masing-masing butuh binary CLI asli + kredensial + checkpoint operator untuk konfirmasi wire contract. Per provider, satu langkah disetujui.
4. **Hapus branch yang sudah merged** — housekeeping opsional. Banyak `feat/acp-*`, `feat/codex-*`, `feat/f1-*`, `feat/f2-*` branch yang sudah merged dan bisa di-prune.
5. **Curl import feature follow-up:** `internal/server/curl_import.go` sudah di-land sebagai hot-fix (commit `17322f6`) untuk repair build. Implementasi lengkap tapi **belum ada unit test** — gap yang perlu di-address sebelum dipakai luas. Branch `feat/curl-import-connection` punya perubahan tambahan (proxy.go, provider_bootstrap.go) yang masih unmerged; rebase/merge setelah versi stabil.

---

## 10. Next Checkpoint

**Codex Full M5 Live Validation** — begitu operator menyediakan kredensial valid:

```bash
LINTASAN_CODEX_LIVE=1 \
LINTASAN_CODEX_ACP_BIN=/tmp/codex-acp-install/node_modules/@zed-industries/codex-acp-linux-x64/bin/codex-acp \
OPENAI_API_KEY=sk-...(valid) \
go test ./internal/expprovider/ -run TestCodexLive_AdmissionFlow -v
```

PASS = tool loop close (acceptance penuh) → Codex fully validated. Lapor PASS/FAIL apa adanya.

Cohort-A lain: live validation per provider (real CLI + kredensial), masing-masing checkpoint operator-gated terpisah.

---

## 11. Future Roadmap (high-level, tidak mengikat)

- **Codex:** lengkapi M5 live validation → (opsional) opt-in activation sebagai checkpoint gated terpisah.
- **Cohort A lanjutan:** Claude Code, Gemini CLI, Copilot — descriptor + mapping + fixture sudah siap (readiness); tinggal live validation per provider (konfirmasi wire contract lawan CLI asli) di belakang checkpoint masing-masing.
- **E2 territory:** ganti `EnvCredentialSource` dengan encrypted per-provider store di belakang interface `CredentialSource` yang sama.
- **F2.4:** validasi diskriminatif setelah provider kedua aktif (pool prod saat ini size-1 → enforcement armed tapi inert/drop-nothing).

> Roadmap di luar daftar ini (Cursor, Windsurf, Antigravity, Kiro, browser-backed) **belum di-scope**. Jangan mulai tanpa checkpoint eksplisit.

---

## 12. Hard Constraints / Invariants

1. **Invariant 1 — Opt-in only:** Experimental provider dijangkau HANYA via sinyal eksplisit. Default/auto/smart routing tak pernah memilihnya.
2. **Invariant 2 — Process containment:** agent jalan sebagai E1 subprocess; crash/hang/panic ditangani sebagai contained error, bukan gateway panic.
3. **Invariant 3 — Credential isolation:** adapter TAK PERNAH pegang core credential store. Secret di-inject scoped per-provider ke child env saat launch, tak pernah di-bake ke LaunchSpec.
4. **Invariant 4 — No dark egress:** adapter cuma pegang `CredentialSource` indirection + `LaunchSpec`; secara struktural tak bisa menjangkau `internal/auth`.
5. **Invariant 5 — Declared capabilities tak dipercaya untuk routing:** capability Experimental cuma ditampilkan (dengan risk badge), tak pernah jadi dasar Official routing.
6. **Membrane satu-arah:** `Track()==Experimental` → difilter keluar dari `RoutableProviders`. Dependency `expprovider → provider/experimental` satu-arah; core tak pernah import expprovider.
7. **Acceptance principle (M5):** sebuah turn valid HANYA jika tool loop CLOSE — stream-text-only BUKAN acceptance. Wajib: ≥1 tool call, identifier fidelity, terminal stopReason non-kosong.
8. **Worktree isolation wajib:** main worktree `~/lintasan-go` di-scrub proses git eksternal (file untracked terhapus ~60s, ref drift). SELALU kerja di isolated worktree (`git worktree add -b feat/x /tmp/x main`), commit di branch, diff vs `HEAD~1`/`main`, BUKAN andalkan untracked di main.
9. **Onboarding = descriptor + mapping + fixture:** menambah Experimental provider DILARANG mengubah `framework.go`. Provider baru masuk sebagai `ProviderDescriptor` + credential mapping + fixture. (Framework adalah baseline sejak `52647d1`.)

---

## 13. Do-Not Rules

- ❌ **Jangan** wire Experimental provider ke proxy hot path / production router.
- ❌ **Jangan** flip flag aktivasi / deploy / auto-activate sebagai bagian dari onboarding. Onboarding berakhir di verdict GO + registrasi dormant. Aktivasi = checkpoint gated terpisah.
- ❌ **Jangan** percaya `git describe HEAD` untuk menyimpulkan apa yang di binary prod — verifikasi SHA `/proc/<pid>/exe` vs on-disk (binary bisa tertinggal di belakang main, seperti sekarang: 5 commit).
- ❌ **Jangan** bake secret ke `LaunchSpec.BaseEnv` (langgar Invariant 3).
- ❌ **Jangan** buat fixture permissive — fixture WAJIB strict (enforce auth-order, sessionId, ContentBlock, cwd) supaya menangkap wire regression. (Pelajaran live validation Codex.)
- ❌ **Jangan** percaya descriptor Cohort-A yang UNVERIFIED sebagai fakta — wire contract asli WAJIB dikonfirmasi lawan CLI asli sebelum dipercaya/diaktifkan.
- ❌ **Jangan** ubah `framework.go` untuk menambah provider (langgar Invariant 9).
- ❌ **Jangan** palsukan synthetic evidence/key/topology untuk memaksa checkpoint maju. Bedakan validation-blocker vs safety-blocker; lapor jujur.
- ❌ **Jangan** campur Shape-1 (Responses ingress, `feat/codex-m0-skeleton`) dengan Shape-2 (ACP subprocess provider).
- ❌ **Jangan** merge/deploy/aktivasi tanpa aba-aba operator.

---

## 14. Recovery / Rollback Notes

- **main saat ini:** `17322f6`. Tag `v0.24.0` di `cfc007a`. 1 commit sejak tag = `17322f6` (curl-import hot-fix). Sebelum versi reset: `6b29d79` (HEAD terakhir dengan `v2.4.0`). Baseline sebelum framework: `4794542` (Codex remediation). Sebelum Codex: `0a6bbd6` (ACP reconciliation). Sebelum Experimental sama sekali: `ca85973`.
- **Rollback branch belum-merge:** cukup jangan merge. Kalau sudah merge & belum deploy: `git revert <sha>` (additive-only, clean) atau (karena FF) `git reset --hard <prev>` + force-push (destruktif, butuh izin operator).
- **Prod rollback:** prod binary sekarang = main (`17322f6`). Untuk roll back ke state pre-Experimental (binary yang **tidak** memuat Experimental sama sekali), target = `ca85973` — binary lama sudah di-overwrite, jadi perlu re-build dari commit itu (`git checkout ca85973 && make build` + deploy). Catatan: rollback ini berarti kehilangan curl-import endpoint (commit `17322f6`) — pastikan itu acceptable dulu.
- **F2.4 disarm:** set `capability_enforce_enabled=false` di prod DB `data/lintasan.db` (flag OFF = inert). Tak perlu rebuild — dibaca saat bootstrap; restart service untuk reload.
- **Backup deploy DB:** `~/backups/lintasan-*-deploy-*/lintasan.db` (snapshot per deploy).
- **Build/deploy:** `make build` → `sudo systemctl stop lintasan` → swap binary → `start` → `curl localhost:20180/health` (cek versi). Downtime ~0.2–0.3s. Detail di `AGENTS.md §3`.
- **PENTING: pre-deploy build hygiene.** Main worktree `~/lintasan-go` di-scrub proses git eksternal (file untracked KEHAPUS ~60s, ref drift). Sebelum build, **WAJIB `git status --porcelain` harus kosong**. Parkir file untracked ke `/tmp/lintasan-parked-<n>/` (jangan dihapus — `.bak`/`.new` = rollback artifact). Setelah build selesai, restore. Kalau skip step ini → binary vcs.modified=true (degraded provenance) atau file hilang saat deploy.

---

*Akhir AGENT.md. Reconciled terhadap git + runtime 2026-06-05 (`main @ 17322f6`, prod binary `7b273520` / `v0.24.0-1-g17322f6`). Kalau ada yang berubah di repo/prod, update file ini lebih dulu sebelum lanjut checkpoint.*
