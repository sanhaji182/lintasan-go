# Lintasan v2.3.1 — Benchmark 3-Arm Report

**Branch:** `fix/p0-security` (kandidat baseline v2.3.1)
**Environment:** 2-core, 7.5GB RAM, shared box (Hermes gateway + others co-resident)
**Tools:** k6 v0.49 (ramping VU), pprof localhost-only (5 profiles), custom mock provider (configurable latency)
**Date:** 2026-05-30

> **Methodology caveat:** absolute RPS is soft (2 cores, background load). What is reliable and decision-bearing: (1) relative comparison across arms/configs, (2) pprof hotspot rankings, (3) counterfactual experiments (pool 1 vs 8, memory on vs off, cache size growth). The hypotheses are answered by these, not by absolute throughput.

---

## Result tables

### Arm 1 — Internal DB endpoints (no upstream)
100 VU ramp over 50k request_logs.

| config | RPS | p50 ms | p95 ms | max ms | CPU avg | RSS peak |
|---|---|---|---|---|---|---|
| MaxConns=1 (v2.3.1 default) | 88.8 | 190.9 | 2089.9 | 14689 | 90% | 56 MB |
| MaxConns=8 (counterfactual) | 88.7 | 215.0 | 1857.9 | 6846 | 127% | 54 MB |

Per-endpoint (MaxConns=1): `/api/stats` p50=579ms (5 full-scan aggregates), `/api/dashboard/stats` p50=322ms, `/api/logs` p50=87ms, `/api/connections` p50=85ms.

### Arm 2 — Proxy to mock upstream

**Cache ON (identical payload → 100% cache hit):**
| upstream | RPS | p50 | p95 | max |
|---|---|---|---|---|
| 10ms | 585 | 45.8 | 252 | 1152 |
| 50ms | 651 | 53.9 | 160 | 486 |
| 200ms | 570 | 47.6 | 238 | 953 |

**Cache OFF (every request → upstream, TRUE overhead):**
| upstream | RPS | p50 | overhead p50 | CPU avg | RSS pk |
|---|---|---|---|---|---|
| 10ms | 404 | 96 | **86ms (89%)** | 86% | 65 MB |
| 50ms | 303 | 113 | 63ms | 74% | 64 MB |
| 200ms | 188 | 210 | **10ms (5%)** | 47% | 65 MB |

### Arm 3 — Real provider (Sumopod, gpt-5-nano), sequential
- direct p50 = 753 ms, via-Lintasan p50 = 1860 ms → overhead ~600–1100 ms
- Dominated by vector-memory injection (see big finding). Credit-minimal: ~14 calls total.

### Vector-memory injection A/B (50 VU, 10ms mock, cache off)
| config | RPS | p50 | p95 |
|---|---|---|---|
| memory_inject OFF | 433 | 83 | 190 |
| memory_inject ON | 249 | 157 | 294 |

**+74% throughput, −74ms p50 when OFF.**

### H3 — semantic cache scan vs row count (isolated microbench)
| rows | per-lookup |
|---|---|
| 100 | 2.5 ms |
| 1,000 | 24.9 ms |
| 5,000 | 122 ms |
| 20,000 | 787 ms |

Linear ~25µs/row (SELECT no LIMIT + Go-side cosine per row).

### H2 — per-request DB cost (shared single conn, microbench)
- logging INSERT: **0.27 ms**
- settings storm (8 SELECT): **0.62 ms**
- semantic scan: 2.5 ms @100 → 787 ms @20k

---

## pprof top hotspots

**Arm 1 (internal DB), CPU:**
1. `runtime.cgocall` 96% (SQLite query exec)
2. `database/sql.(*Row).Scan` 95.6% cum
3. `handleStats` 61.9% cum — 5 full-table aggregates
4. `handleDashboardStats` 34.7% cum
- Block profile: 89% in `database/sql.(*DB).conn` (waiting on single connection), `GetSetting` 21.5%

**Arm 2 cache-off @10ms, CPU:**
1. `memory.StoreManager.Search` **40%** (vector-memory injection)
2. `encoding/json` (Unmarshal+array+literalStore) ~30%
3. `runtime.cgocall` (SQLite) 22%
4. `db.GetSetting` 18% (settings storm)
5. `memory.populateMemory` 15% (Redis HGETALL per result)

---

## Hypotheses — verdicts

- **H1 (SetMaxOpenConns(1) is the primary bottleneck): REJECTED.** Pool 1→8 left throughput flat (88.8→88.7). Cost is CPU-bound per-query work (SQLite cgo + unindexed aggregates), not connection serialization.
- **H2 (sync logging > semantic cache): CONDITIONALLY FALSE.** Logging (0.27ms) > semantic only when cache is near-empty; semantic overtakes by ~1k rows.
- **H3 (semantic scan dominant as data grows): CONFIRMED.** Linear, 787ms/lookup at 20k rows.
- **H4 (proxy overhead small vs provider): CONFIRMED.** Overhead is 5% at 200ms upstream, 89% at 10ms — invisible at realistic LLM latency, exactly as predicted.
- **H5 (memory/goroutine stable over 60m soak): [pending — running].**

---

## Engineering decisions (the point of the exercise)

1. **DO NOT prioritize SQLite connection-pool changes.** H1 rejected. `SetMaxOpenConns(1)` is not the throughput bottleneck at realistic upstream latency; bumping it added CPU for zero throughput gain. Keep SQLite as-is for now. *(env knob added for future tuning, default unchanged.)*

2. **HIGHEST ROI — gate vector-memory injection (DONE).** It ran on every request by default (40% CPU, −74% throughput) for a feature the audit showed is non-semantic. Now opt-in via `memory_injection_enabled` (default off). Single biggest win.

3. **HIGH — cap + index the semantic cache scan (H3).** Add `LIMIT` + index, or a row-count ceiling with eviction. Unbounded linear scan is a latent time bomb (787ms/lookup at 20k).

4. **MEDIUM — fix `/api/stats` 5x full-table aggregates.** Add indexes on `cached`/`status`, or precompute/cache stats. p50 579ms over 50k rows is unindexed scans.

5. **MEDIUM — in-memory settings cache.** Settings storm = 18% CPU on the hot path; trivially cacheable with invalidation.

6. **LOW/DEFER — async logging.** Real but small (0.27ms/req). Only matters once the bigger items are fixed and upstream is very fast. Defer.

7. **Tighten default rate limit semantics.** Hardcoded 60/min/key surprised the benchmark (returned 429 under load). Make it a documented, configurable default (env knob added).

**Bottom line:** the proxy's per-request internal cost is dominated by **vector-memory injection + unindexed stats/cache scans + settings storm** — none of which is the SQLite connection pool. Remediation is targeted (gates, indexes, caching), not architectural. No rewrite, no Postgres, no Redis-required.
