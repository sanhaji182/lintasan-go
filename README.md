# Lintasan Go

High-performance LLM proxy router. Rewrite of [Lintasan](https://github.com/sanhaji182/lintasan) in Go.

## Why Go?

| | Node.js (v1) | Go (v2) |
|---|---|---|
| RAM | ~500MB | ~14MB |
| Binary | ~200MB (node_modules) | 13MB single binary |
| Startup | ~2s | ~50ms |
| Concurrent connections | ~10,000 | ~50,000+ |

**35x less RAM, single binary, instant startup.**

## Features

- OpenAI-compatible API (`/v1/chat/completions`, `/v1/models`, `/v1/embeddings`)
- Multi-provider proxy with priority routing
- Streaming support (SSE)
- SQLite embedded database (zero config)
- Connection management API
- Request logging & analytics
- CORS support
- Master key authentication
- CLI with cobra (`lintasan start`, `lintasan setup`, `lintasan mitm start`)
- Compatible with existing Lintasan Node.js database schema

## Quick Start

```bash
# Build
go build -o bin/lintasan ./cmd/lintasan/

# Run
./bin/lintasan start

# Custom port
PORT=8080 ./bin/lintasan start
```

## API Endpoints

### OpenAI Compatible
- `POST /v1/chat/completions` — Proxy chat completions
- `POST /v1/embeddings` — Proxy embeddings
- `GET /v1/models` — List available models

### Management
- `GET /health` — Health check
- `GET /api/connections` — List connections
- `POST /api/connections` — Add connection
- `DELETE /api/connections/{id}` — Remove connection
- `GET /api/combos` — List combos
- `POST /api/combos` — Create combo
- `GET /api/stats` — Request statistics
- `GET /api/settings` — Get settings
- `PUT /api/settings` — Update settings

## Configuration

Environment variables:
- `PORT` — Server port (default: 20180)
- `LINTASAN_DATA_DIR` — Data directory (default: ./data)
- `LINTASAN_MASTER_KEY` — Master API key (overrides DB setting)
- `MITM_PORT` — MITM bridge port (default: 8443)

## License

MIT
