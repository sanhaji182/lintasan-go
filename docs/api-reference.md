# Lintasan Go API Reference — SvelteKit Frontend

## Auth
- All `/api/*` need `Authorization: Bearer <master_key>` except:
  - `/api/auth/*`, `/api/dashboard/*`, `/health`
- CORS: `Access-Control-Allow-Origin: *`
- Login: POST `/api/auth/login` → `{"success":true,"token":"dashboard-session"}`
- Check: GET `/api/auth/check` → `{"authenticated":true}`

## API Endpoints (data wrapped in {"data": ...})

### Stats & Overview
- GET `/api/stats` → {totalRequests, cachedRequests, cacheHitRate, avgLatency, tokensToday, tokensMonth, tokensSaved, tokensCompressed, activeModels, activeConnections, features[], providers[], requestVolume[]}
- GET `/api/dashboard/stats` → {total_requests, active_connections, cache_hit_rate, avg_latency, uptime}

### Connections
- GET `/api/connections` → {data: [{id, name, base_url, api_key(masked), format, is_active, priority, models_count, created_at}]}
- POST `/api/connections` → create
- PATCH `/api/connections` → toggle active
- DELETE `/api/connections/{id}` → delete
- POST `/api/connections/test` → test connection
- GET `/api/connections/sync` → sync all
- POST `/api/models/sync/{connection_id}` → sync single

### Routing (Combos)
- GET `/api/combos` → {data: [...]}
- POST `/api/combos` → create
- PUT `/api/combos?id=` → update
- DELETE `/api/combos?id=` → delete
- POST `/api/routing/reorder` → reorder

### Load Balancer
- GET `/api/load-balancer` → {data: {strategy: "..."}}
- POST `/api/load-balancer` → set strategy

### Aliases
- GET `/api/aliases` → {data: [...]}
- POST `/api/aliases` → create/update
- DELETE `/api/aliases?id=` → delete

### Fallback
- GET `/api/fallback` → {data: {model_chains, connection_chains, stats}}
- POST `/api/fallback` → create
- DELETE `/api/fallback` → delete

### Logs
- GET `/api/logs?limit=20` → {data: [{id, connection_id, provider, model, status, input_tokens, output_tokens, latency_ms, cached, error, created_at}]}
- NOTE: `/api/logs/stream` does NOT exist — use polling

### Analytics
- GET `/api/analytics` → {tokensSavedToday, cacheHitRate, totalTokensUsed, costSaved, avgLatency, totalRequests, daily[], breakdown}

### Usage
- GET `/api/usage` → {providers[], models[], daily[]}

### Keys
- GET `/api/keys` → {data: [...]}
- POST `/api/keys` → create (action: "create")

### Settings
- GET `/api/settings` → {data: {key: value, ...}}
- PUT `/api/settings` → update

### Providers/Presets
- GET `/api/providers/presets` → {data: [...], categories: [...]}
- GET `/api/providers/presets/config?id=` → single preset
- POST `/api/providers/presets/test` → test preset

### Teams
- GET/POST `/api/teams`
- GET/PUT/DELETE `/api/teams/{id}`
- GET/POST `/api/teams/{id}/members`

### Users
- GET/POST `/api/users`
- GET/PUT/DELETE `/api/users/{id}`

### Webhooks
- GET/POST `/api/webhooks`

### Plugins
- GET/POST `/api/plugins`
- GET/POST `/api/plugins/store`
- POST `/api/plugins/generate`

### Backup
- GET/POST `/api/backup`

### Chat
- POST `/api/chat-test` → proxied to chat completions
- POST `/v1/chat/completions` → SSE streaming (OpenAI format)

### Other
- GET `/api/models/catalog` → catalog with pricing
- GET `/api/models/discovered` → discovered models
- GET/POST `/api/models/manual` → manual model management
- GET `/api/export` → export config
- GET `/api/favicon?domain=` → favicon proxy
