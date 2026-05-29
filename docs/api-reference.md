# 🇮🇩 Lintasan Go — API Reference (SvelteKit Frontend)

> Referensi lengkap semua endpoint API. Semua response dibungkus dalam `{"data": ...}` kecuali dinyatakan lain.

---

# 🇬🇧 Lintasan Go — API Reference (SvelteKit Frontend)

> Complete API endpoint reference. All responses are wrapped in `{"data": ...}` unless stated otherwise.

---

## 🔐 Auth / Autentikasi

<details open>
<summary>🇮🇩</summary>

- Semua `/api/*` memerlukan `Authorization: Bearer YOUR_MASTER_KEY`, kecuali:
  - `/api/auth/*` — endpoint login/check
  - `/api/dashboard/*` — dashboard API (no auth needed)
  - `/health` — health check
  - `/api/providers/presets` — public presets
- CORS: `Access-Control-Allow-Origin: *`
- **Login:** `POST /api/auth/login` → `{"success": true, "token": "dashboard-session"}`
- **Check:** `GET /api/auth/check` → `{"authenticated": true}`

</details>

<details>
<summary>🇬🇧</summary>

- All `/api/*` require `Authorization: Bearer YOUR_MASTER_KEY`, except:
  - `/api/auth/*` — login/check endpoints
  - `/api/dashboard/*` — dashboard API (no auth)
  - `/health` — health check
  - `/api/providers/presets` — public presets
- CORS: `Access-Control-Allow-Origin: *`
- **Login:** `POST /api/auth/login` → `{"success": true, "token": "dashboard-session"}`
- **Check:** `GET /api/auth/check` → `{"authenticated": true}`

</details>

---

## 📊 Stats & Overview

| Method | Endpoint | Response (wrapped in `data`) |
|--------|----------|------------------------------|
| `GET` | `/api/stats` | `{totalRequests, cachedRequests, cacheHitRate, avgLatency, tokensToday, tokensMonth, tokensSaved, tokensCompressed, activeModels, activeConnections, features[], providers[], requestVolume[]}` |
| `GET` | `/api/dashboard/stats` | `{total_requests, active_connections, cache_hit_rate, avg_latency, uptime}` |

---

## 🔗 Connections

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/connections` | List all → `{data: [{id, name, base_url, api_key(masked), format, is_active, priority, models_count, created_at}]}` |
| `POST` | `/api/connections` | Create new connection |
| `PATCH` | `/api/connections` | Toggle active/inactive |
| `DELETE` | `/api/connections/{id}` | Delete connection |
| `POST` | `/api/connections/test` | Test connection (latency + model count) |
| `GET` | `/api/connections/sync` | Sync all connections (discover models) |
| `POST` | `/api/models/sync/{connection_id}` | Sync single connection models |

---

## 🔀 Routing (Combos)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/combos` | List all combos → `{data: [...]}` |
| `POST` | `/api/combos` | Create combo |
| `PUT` | `/api/combos?id={id}` | Update combo |
| `DELETE` | `/api/combos?id={id}` | Delete combo |
| `POST` | `/api/routing/reorder` | Reorder combos |

---

## ⚖ Load Balancer

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/load-balancer` | Get strategy → `{data: {strategy: "round-robin"}}` |
| `POST` | `/api/load-balancer` | Set strategy (`round-robin`, `weighted`, `least-latency`) |

---

## 🏷 Aliases

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/aliases` | List aliases → `{data: [...]}` |
| `POST` | `/api/aliases` | Create/update alias |
| `DELETE` | `/api/aliases?id={id}` | Delete alias |

---

## 🔄 Fallback

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/fallback` | List chains → `{data: {model_chains, connection_chains, stats}}` |
| `POST` | `/api/fallback` | Create fallback chain |
| `DELETE` | `/api/fallback?type={type}&id={id}` | Delete fallback chain |

**Request body (POST):**
```json
{
  "type": "model",
  "id": "gpt4-fallback",
  "fallbacks": ["gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"]
}
```

---

## 📜 Logs

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/logs?limit=20` | Recent request logs → `{data: [{id, connection_id, provider, model, status, input_tokens, output_tokens, latency_ms, cached, error, created_at}]}` |

> ⚠️ `/api/logs/stream` tidak tersedia — gunakan polling

---

## 📈 Analytics

| Method | Endpoint | Response |
|--------|----------|----------|
| `GET` | `/api/analytics` | `{tokensSavedToday, cacheHitRate, totalTokensUsed, costSaved, avgLatency, totalRequests, daily[], breakdown}` |

---

## 📊 Usage

| Method | Endpoint | Response |
|--------|----------|----------|
| `GET` | `/api/usage` | `{providers[], models[], daily[]}` |

---

## 🔑 API Keys

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/keys` | List keys → `{data: [{id, name, key(masked), usage, created_at}]}` |
| `POST` | `/api/keys` | Create key (`action: "create"`) |
| `DELETE` | `/api/keys?id={id}` | Revoke key |

---

## ⚙ Settings

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/settings` | Get all settings → `{data: {key: value, ...}}` |
| `PUT` | `/api/settings` | Update settings (partial) |

---

## 🏪 Providers & Presets

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/providers/presets` | 113 presets → `{data: [...], categories: [...]}` |
| `GET` | `/api/providers/presets/config?id={id}` | Single preset detail |
| `POST` | `/api/providers/presets/test` | Test preset connectivity |

---

## 👥 Teams

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/teams` | List teams |
| `POST` | `/api/teams` | Create team |
| `GET` | `/api/teams/{id}` | Team detail |
| `PUT` | `/api/teams/{id}` | Update team |
| `DELETE` | `/api/teams/{id}` | Delete team |
| `GET` | `/api/teams/{id}/members` | List members |
| `POST` | `/api/teams/{id}/members` | Add member |

---

## 👤 Users

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/users` | List users |
| `POST` | `/api/users` | Create user |
| `GET` | `/api/users/{id}` | User detail |
| `PUT` | `/api/users/{id}` | Update user |
| `DELETE` | `/api/users/{id}` | Delete user |

---

## 🔔 Webhooks

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/webhooks` | List webhooks |
| `POST` | `/api/webhooks` | Create webhook |
| `DELETE` | `/api/webhooks?id={id}` | Delete webhook |

---

## 🧩 Plugins

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/plugins` | List installed plugins |
| `POST` | `/api/plugins` | Enable/disable plugin (`{name: "...", enabled: true/false}`) |
| `GET` | `/api/plugins/store` | Plugin store → `{data: [...]}` |
| `POST` | `/api/plugins/store` | Install from store |
| `POST` | `/api/plugins/generate` | AI-generate plugin |

---

## 💾 Backup

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/backup` | List backup files |
| `POST` | `/api/backup` | Create/restore backup |
| `DELETE` | `/api/backup?file={filename}` | Delete backup file |

---

## 💬 Chat & LLM

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/chat/completions` | OpenAI-compatible chat (SSE streaming) |
| `POST` | `/api/chat-test` | Test chat (proxied) |

---

## 🧠 Vector Memory

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/memory` | Store memory |
| `GET` | `/v1/memory/search?q={query}&limit=5` | Search memories |
| `GET` | `/v1/memory/stats` | Memory stats |

---

## 🎨 Media (Image + Audio)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/images/generations` | Image generation (DALL-E / SD) |
| `POST` | `/v1/audio/speech` | Text-to-speech (TTS) |
| `POST` | `/v1/audio/transcriptions` | Speech-to-text (STT) |

---

## 🌐 Web Search

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/web/search` | Augmented chat with web results |

---

## 🔍 Models

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/models` | OpenAI-compatible model list |
| `GET` | `/api/models/catalog` | Model catalog with pricing |
| `GET` | `/api/models/discovered` | Auto-discovered models |
| `GET` | `/api/models/manual` | Manually added models |
| `POST` | `/api/models/manual` | Add manual model |

---

## 🏷 Other

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/export` | Export config JSON |
| `GET` | `/api/favicon?domain={domain}` | Proxy favicon |
| `GET` | `/api/providers/discover` | Scan for free providers |
| `GET` | `/health` | Health check → `{"status":"ok","version":"2.0.0","uptime":"..."}` |

---

## 📝 Response Format

Semua response JSON mengikuti format:

```json
{
  "data": {
    // response content
  }
}
```

Kecuali endpoint `/health`, `/v1/chat/completions` (SSE streaming), dan endpoint raw lainnya.

---

## 🚨 Error Codes

| Code | 🇮🇩 Indonesia | 🇬🇧 English |
|------|-------------|------------|
| `400` | Request tidak valid | Invalid request |
| `401` | API key tidak valid | Invalid API key |
| `404` | Endpoint/model tidak ditemukan | Endpoint/model not found |
| `429` | Rate limit atau kuota habis | Rate limited or quota exhausted |
| `500` | Internal server error | Internal server error |
| `502` | Semua provider gagal (termasuk daftar yang dicoba) | All providers failed (with retry list) |
