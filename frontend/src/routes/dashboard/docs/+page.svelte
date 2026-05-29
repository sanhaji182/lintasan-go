<script lang="ts">
  import {
    BookOpen, ChevronRight, ChevronDown, Search,
    Zap, Code2, Settings, Lightbulb, Copy, Check, Globe
  } from 'lucide-svelte';

  interface DocSubsection {
    id: string;
    id_title: string;
    en_title: string;
    id_content: string;
    en_content: string;
    id_code?: string;
    en_code?: string;
    language?: string;
  }

  interface DocSection {
    id: string;
    id_title: string;
    en_title: string;
    icon: typeof BookOpen;
    subsections: DocSubsection[];
  }

  let lang = $state<'id' | 'en'>('en');
  let activeSection = $state('getting-started');
  let searchQuery = $state('');
  let expandedSections = $state<Set<string>>(new Set(['getting-started']));
  let copiedCode = $state<string | null>(null);

  const docs: DocSection[] = [
    {
      id: 'getting-started',
      id_title: 'Mulai Cepat',
      en_title: 'Getting Started',
      icon: Zap,
      subsections: [
        {
          id: 'quick-start',
          id_title: 'Mulai Cepat',
          en_title: 'Quick Start',
          id_content: `Lintasan adalah **LLM proxy gateway** yang berada di antara aplikasi kamu dan provider AI. Dia menangani routing, caching, rate limiting, dan observability secara otomatis.

**🇮🇩 Satu endpoint untuk semua provider.** Tidak perlu ganti SDK atau kelola banyak API key.`,
          en_content: `Lintasan is an **LLM proxy gateway** that sits between your application and AI providers. It handles routing, caching, rate limiting, and observability automatically.

**🇬🇧 One endpoint for all providers.** No need to switch SDKs or manage multiple API keys.`,
          id_code: `# Download binary (single file, 24MB)
curl -L -o lintasan \\
  https://github.com/sanhaji182/lintasan-go/releases/latest/download/lintasan
chmod +x lintasan

# Jalankan server
./lintasan start

# Server berjalan di http://localhost:20180
# Dashboard di http://localhost:20180/dashboard`,
          en_code: `# Download binary (single file, 24MB)
curl -L -o lintasan \\
  https://github.com/sanhaji182/lintasan-go/releases/latest/download/lintasan
chmod +x lintasan

# Start the server
./lintasan start

# Server running at http://localhost:20180
# Dashboard at http://localhost:20180/dashboard`,
          language: 'bash',
        },
        {
          id: 'providers',
          id_title: 'Koneksi Provider',
          en_title: 'Connecting Providers',
          id_content: `Tambah provider lewat **dashboard UI** atau **API**. 113 provider presets sudah siap — tinggal isi API key.

**Provider yang didukung:** OpenAI, Anthropic, DeepSeek, Google Gemini, Groq, Mistral, xAI, dan 100+ lainnya via LiteLLM presets.`,
          en_content: `Add providers via the **dashboard UI** or **API**. 113 provider presets are ready — just add your API key.

**Supported providers:** OpenAI, Anthropic, DeepSeek, Google Gemini, Groq, Mistral, xAI, and 100+ more via LiteLLM presets.`,
          id_code: `# Tambah koneksi via API
curl -X POST http://localhost:20180/api/connections \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "openai",
    "base_url": "https://api.openai.com/v1",
    "api_key": "sk-...",
    "models": ["gpt-4o", "gpt-4o-mini"]
  }'

# Atau: buka dashboard → Accounts → Add Provider`,
          en_code: `# Add connection via API
curl -X POST http://localhost:20180/api/connections \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "openai",
    "base_url": "https://api.openai.com/v1",
    "api_key": "sk-...",
    "models": ["gpt-4o", "gpt-4o-mini"]
  }'

# Or: open dashboard → Accounts → Add Provider`,
          language: 'bash',
        },
        {
          id: 'first-request',
          id_title: 'Request Pertama',
          en_title: 'Your First Request',
          id_content: `Setelah gateway berjalan dan provider tersambung, kirim request menggunakan **API OpenAI-compatible**:`,
          en_content: `Once the gateway is running and a provider is connected, send requests using the **OpenAI-compatible API**:`,
          id_code: `curl http://localhost:20180/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_MASTER_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "Halo! Apa kabar?"}
    ]
  }'`,
          en_code: `curl http://localhost:20180/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_MASTER_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "Hello! How are you?"}
    ]
  }'`,
          language: 'bash',
        },
      ],
    },
    {
      id: 'api-reference',
      id_title: 'Referensi API',
      en_title: 'API Reference',
      icon: Code2,
      subsections: [
        {
          id: 'chat-completions',
          id_title: 'Chat Completions',
          en_title: 'Chat Completions',
          id_content: `Endpoint chat completions **sepenuhnya kompatibel dengan OpenAI API**.

**Endpoint:** \`POST /v1/chat/completions\`

**Parameter:**
- \`model\` (string, wajib) — ID model atau nama combo
- \`messages\` (array, wajib) — Array pesan (system/user/assistant)
- \`temperature\` (number) — Suhu sampling (0-2)
- \`max_tokens\` (integer) — Maksimum token output
- \`stream\` (boolean) — Aktifkan streaming (SSE)
- \`top_p\` (number) — Nucleus sampling`,
          en_content: `The chat completions endpoint is **fully compatible with the OpenAI API specification**.

**Endpoint:** \`POST /v1/chat/completions\`

**Parameters:**
- \`model\` (string, required) — Model identifier or combo name
- \`messages\` (array, required) — Array of message objects (system/user/assistant)
- \`temperature\` (number) — Sampling temperature (0-2)
- \`max_tokens\` (integer) — Maximum tokens to generate
- \`stream\` (boolean) — Enable streaming (SSE)
- \`top_p\` (number) — Nucleus sampling`,
          id_code: `{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "Kamu asisten yang membantu."},
    {"role": "user", "content": "Jelaskan quantum computing."}
  ],
  "temperature": 0.7,
  "max_tokens": 1000,
  "stream": false
}`,
          en_code: `{
  "model": "gpt-4o",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Explain quantum computing."}
  ],
  "temperature": 0.7,
  "max_tokens": 1000,
  "stream": false
}`,
          language: 'json',
        },
        {
          id: 'embeddings',
          id_title: 'Embeddings',
          en_title: 'Embeddings',
          id_content: `Generate vector embeddings untuk teks.

**Endpoint:** \`POST /v1/embeddings\`

**Parameter:**
- \`model\` (string, wajib) — Model embedding
- \`input\` (string | array, wajib) — Teks yang di-embed
- \`encoding_format\` (string) — "float" atau "base64"`,
          en_content: `Generate vector embeddings for text inputs.

**Endpoint:** \`POST /v1/embeddings\`

**Parameters:**
- \`model\` (string, required) — Embedding model identifier
- \`input\` (string | array, required) — Text to embed
- \`encoding_format\` (string) — "float" or "base64"`,
          id_code: `curl http://localhost:20180/v1/embeddings \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model": "text-embedding-3-small", "input": "Halo dunia"}'`,
          en_code: `curl http://localhost:20180/v1/embeddings \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model": "text-embedding-3-small", "input": "Hello world"}'`,
          language: 'bash',
        },
        {
          id: 'images-audio',
          id_title: 'Gambar & Audio',
          en_title: 'Images & Audio',
          id_content: `Lintasan juga mem-proxy **image generation** dan **audio (TTS + STT)** via endpoint OpenAI-compatible.

| Endpoint | Fungsi |
|----------|--------|
| \`POST /v1/images/generations\` | Generate gambar (DALL-E / SD) |
| \`POST /v1/audio/speech\` | Text-to-speech |
| \`POST /v1/audio/transcriptions\` | Speech-to-text |
| \`GET /v1/models\` | List semua model tersedia |
| \`GET /v1/memory/search?q=...\` | Vector memory search |`,
          en_content: `Lintasan also proxies **image generation** and **audio (TTS + STT)** via OpenAI-compatible endpoints.

| Endpoint | Function |
|----------|----------|
| \`POST /v1/images/generations\` | Generate images (DALL-E / SD) |
| \`POST /v1/audio/speech\` | Text-to-speech |
| \`POST /v1/audio/transcriptions\` | Speech-to-text |
| \`GET /v1/models\` | List all available models |
| \`GET /v1/memory/search?q=...\` | Vector memory search |`,
          id_code: `# Generate gambar
curl http://localhost:20180/v1/images/generations \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model": "dall-e-3", "prompt": "Matahari terbenam di atas gunung"}'

# Text-to-speech
curl http://localhost:20180/v1/audio/speech \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model": "tts-1", "input": "Halo dunia", "voice": "alloy"}'`,
          en_code: `# Image generation
curl http://localhost:20180/v1/images/generations \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model": "dall-e-3", "prompt": "Sunset over mountains"}'

# Text-to-speech
curl http://localhost:20180/v1/audio/speech \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model": "tts-1", "input": "Hello world", "voice": "alloy"}'`,
          language: 'bash',
        },
      ],
    },
    {
      id: 'features',
      id_title: 'Fitur & Optimasi',
      en_title: 'Features & Optimization',
      icon: Settings,
      subsections: [
        {
          id: 'smart-routing',
          id_title: 'Smart Routing',
          en_title: 'Smart Routing',
          id_content: `Lintasan menggunakan **multi-stage routing** untuk mengarahkan request ke provider terbaik:

1. **Header-based** — \`X-Connection: <id>\` override manual
2. **Model name match** — cocokkan model ke provider
3. **Load-balanced pick** — distribusi berbobot
4. **Priority sort** — provider prioritas tertinggi
5. **Fallback chain** — auto-failover jika gagal`,
          en_content: `Lintasan uses **multi-stage routing** to direct requests to the best provider:

1. **Header-based** — \`X-Connection: <id>\` manual override
2. **Model name match** — match model to provider
3. **Load-balanced pick** — weighted distribution
4. **Priority sort** — highest priority provider first
5. **Fallback chain** — auto-failover on failure`,
        },
        {
          id: 'circuit-breaker',
          id_title: 'Circuit Breaker & Caching',
          en_title: 'Circuit Breaker & Caching',
          id_content: `**Circuit Breaker:** Provider yang gagal auto-disabled selama 30 detik. Ini mencegah request lambat terus-menerus ke provider yang down.

**Settings Cache:** Semua settings di-cache di memory dengan 5-detik TTL. Nol DB read per request — akses instant.

**Fallback Chains:** Definisikan urutan model fallback. Kalau \`gpt-4o\` gagal, auto-coba \`claude-sonnet-4\`, lalu \`gemini-pro\`.`,
          en_content: `**Circuit Breaker:** Failing providers are auto-disabled for 30 seconds. This prevents slow requests to down providers.

**Settings Cache:** All settings are cached in-memory with a 5-second TTL. Zero DB reads per request — instant access.

**Fallback Chains:** Define model fallback order. If \`gpt-4o\` fails, auto-try \`claude-sonnet-4\`, then \`gemini-pro\`.`,
          id_code: `# Buat fallback chain via API
curl -X POST http://localhost:20180/api/fallback \\
  -H "Content-Type: application/json" \\
  -d '{
    "type": "model",
    "id": "production-fallback",
    "fallbacks": ["gpt-4o", "claude-sonnet-4", "gemini-2.5-pro"]
  }'`,
          en_code: `# Create fallback chain via API
curl -X POST http://localhost:20180/api/fallback \\
  -H "Content-Type: application/json" \\
  -d '{
    "type": "model",
    "id": "production-fallback",
    "fallbacks": ["gpt-4o", "claude-sonnet-4", "gemini-2.5-pro"]
  }'`,
          language: 'bash',
        },
        {
          id: 'plugins',
          id_title: 'Plugin System',
          en_title: 'Plugin System',
          id_content: `Lintasan punya **plugin system** yang bisa di-extend tanpa ubah core:

- **Built-in plugins:** Request Logger, Rate Limiter, Cost Guard
- **Plugin Store:** Install plugin dari komunitas
- **AI Generator:** Generate plugin baru dengan natural language

Semua plugin auto-register dan jalan di request pipeline.`,
          en_content: `Lintasan has an extensible **plugin system** that works without core changes:

- **Built-in plugins:** Request Logger, Rate Limiter, Cost Guard
- **Plugin Store:** Install community plugins
- **AI Generator:** Generate new plugins with natural language

All plugins auto-register and run in the request pipeline.`,
        },
        {
          id: 'load-balancer',
          id_title: 'Load Balancer & Aliases',
          en_title: 'Load Balancer & Aliases',
          id_content: `**Load Balancer:** Distribusikan request ke beberapa provider dengan strategi:

- \`round-robin\` — Giliran merata
- \`weighted\` — Berdasarkan bobot prioritas
- \`least-latency\` — Provider tercepat

**Aliases:** Buat nama pendek untuk model — \`gpt4\` → \`gpt-4o\`.`,
          en_content: `**Load Balancer:** Distribute requests across providers with strategies:

- \`round-robin\` — Even distribution
- \`weighted\` — Based on priority weights
- \`least-latency\` — Fastest provider

**Aliases:** Create short names for models — \`gpt4\` → \`gpt-4o\`.`,
        },
        {
          id: 'monitoring',
          id_title: 'Monitoring & Observability',
          en_title: 'Monitoring & Observability',
          id_content: `Dashboard 17 halaman untuk monitoring real-time:

- **Overview:** Request, token, cache hit rate, latency
- **Logs:** Request log dengan filter & search
- **Analytics:** Token savings, cache performance
- **Usage:** Breakdown per provider/model
- **Cost Tracking:** Real-time biaya per request`,
          en_content: `17-page dashboard for real-time monitoring:

- **Overview:** Requests, tokens, cache hit rate, latency
- **Logs:** Filterable & searchable request log
- **Analytics:** Token savings, cache performance
- **Usage:** Per-provider/model breakdown
- **Cost Tracking:** Real-time cost per request`,
        },
      ],
    },
    {
      id: 'examples',
      id_title: 'Contoh Kode',
      en_title: 'Code Examples',
      icon: Lightbulb,
      subsections: [
        {
          id: 'nodejs-client',
          id_title: 'Node.js / TypeScript',
          en_title: 'Node.js / TypeScript',
          id_content: `Gunakan OpenAI SDK yang diarahkan ke Lintasan gateway. **Drop-in replacement** — ganti \`baseURL\` aja.`,
          en_content: `Use the OpenAI SDK pointed at your Lintasan gateway. **Drop-in replacement** — just change the \`baseURL\`.`,
          id_code: `import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:20180/v1',
  apiKey: process.env.LINTASAN_API_KEY,
});

// Chat completion
const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Halo!' }],
});

console.log(response.choices[0].message.content);`,
          en_code: `import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:20180/v1',
  apiKey: process.env.LINTASAN_API_KEY,
});

// Chat completion
const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Hello!' }],
});

console.log(response.choices[0].message.content);`,
          language: 'typescript',
        },
        {
          id: 'streaming',
          id_title: 'Streaming (SSE)',
          en_title: 'Streaming (SSE)',
          id_content: `Streaming response real-time dengan Server-Sent Events:`,
          en_content: `Real-time streaming responses with Server-Sent Events:`,
          id_code: `const stream = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Tulis puisi.' }],
  stream: true,
});

for await (const chunk of stream) {
  const content = chunk.choices[0]?.delta?.content;
  if (content) process.stdout.write(content);
}`,
          en_code: `const stream = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Write a poem.' }],
  stream: true,
});

for await (const chunk of stream) {
  const content = chunk.choices[0]?.delta?.content;
  if (content) process.stdout.write(content);
}`,
          language: 'typescript',
        },
        {
          id: 'python-client',
          id_title: 'Python',
          en_title: 'Python',
          id_content: `Gunakan library OpenAI Python. Semua fitur kompatibel:`,
          en_content: `Use the OpenAI Python library. All features compatible:`,
          id_code: `from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:20180/v1",
    api_key="lintasan-master-key",
)

# Chat completion
response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Halo!"}],
)

print(response.choices[0].message.content)

# Embeddings
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="Halo dunia",
)

# Streaming
stream = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Halo!"}],
    stream=True,
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")`,
          en_code: `from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:20180/v1",
    api_key="lintasan-master-key",
)

# Chat completion
response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello!"}],
)

print(response.choices[0].message.content)

# Embeddings
emb = client.embeddings.create(
    model="text-embedding-3-small",
    input="Hello world",
)

# Streaming
stream = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Hello!"}],
    stream=True,
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")`,
          language: 'python',
        },
        {
          id: 'curl-examples',
          id_title: 'curl / Terminal',
          en_title: 'curl / Terminal',
          id_content: `Semua endpoint bisa diakses via curl. Autentikasi pakai \`Authorization: Bearer\` header:`,
          en_content: `All endpoints accessible via curl. Authenticate with \`Authorization: Bearer\` header:`,
          id_code: `# Chat
curl http://localhost:20180/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Halo"}]}'

# Streaming
curl -N http://localhost:20180/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Halo"}],"stream":true}'

# List models
curl http://localhost:20180/v1/models \\
  -H "Authorization: Bearer YOUR_KEY"

# Health check
curl http://localhost:20180/health`,
          en_code: `# Chat
curl http://localhost:20180/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}]}'

# Streaming
curl -N http://localhost:20180/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}],"stream":true}'

# List models
curl http://localhost:20180/v1/models \\
  -H "Authorization: Bearer YOUR_KEY"

# Health check
curl http://localhost:20180/health`,
          language: 'bash',
        },
        {
          id: 'ide-tools',
          id_title: 'AI Coding Tools (IDE/CLI)',
          en_title: 'AI Coding Tools (IDE/CLI)',
          id_content: `Lintasan kompatibel dengan **semua AI coding tools** yang support OpenAI-compatible API. Cukup arahkan ke Lintasan endpoint — otomatis dapat routing, caching, dan fallback.

Berikut konfigurasi untuk tools populer:`,
          en_content: `Lintasan is compatible with **all AI coding tools** that support OpenAI-compatible API. Just point to your Lintasan endpoint — you automatically get routing, caching, and fallback.

Here are configurations for popular tools:`,
        },
        {
          id: 'claude-code',
          id_title: 'Claude Code (Anthropic)',
          en_title: 'Claude Code (Anthropic)',
          id_content: `Claude Code adalah AI coding agent CLI dari Anthropic. Arahkan ke Lintasan dengan environment variable. **Semua model provider bisa dipakai** — tidak terbatas Claude saja.`,
          en_content: `Claude Code is Anthropic's AI coding agent CLI. Point it to Lintasan with environment variables. **Any provider model works** — not limited to Claude.`,
          id_code: `# ~/.bashrc atau ~/.zshrc
export ANTHROPIC_BASE_URL="http://localhost:20180/v1"
export ANTHROPIC_API_KEY="lintasan-master-key"

# Claude Code otomatis pakai Lintasan
claude`,
          en_code: `# ~/.bashrc or ~/.zshrc
export ANTHROPIC_BASE_URL="http://localhost:20180/v1"
export ANTHROPIC_API_KEY="lintasan-master-key"

# Claude Code automatically uses Lintasan
claude`,
          language: 'bash',
        },
        {
          id: 'codex',
          id_title: 'Codex CLI (OpenAI)',
          en_title: 'Codex CLI (OpenAI)',
          id_content: `Codex adalah coding agent CLI dari OpenAI. Support OpenAI-compatible API. **Gunakan model apapun** — GPT, Claude, Gemini, DeepSeek, Groq, dll lewat Lintasan.`,
          en_content: `Codex is OpenAI's coding agent CLI. Supports OpenAI-compatible API. **Use any model** — GPT, Claude, Gemini, DeepSeek, Groq, etc. via Lintasan.`,
          id_code: `export OPENAI_API_KEY="lintasan-master-key"
export OPENAI_API_BASE_URL="http://localhost:20180/v1"

codex edit "Refactor this file to use async/await"`,
          en_code: `export OPENAI_API_KEY="lintasan-master-key"
export OPENAI_API_BASE_URL="http://localhost:20180/v1"

codex edit "Refactor this file to use async/await"`,
          language: 'bash',
        },
        {
          id: 'hermes-agent',
          id_title: 'Hermes Agent',
          en_title: 'Hermes Agent',
          id_content: `Hermes Agent adalah AI assistant dengan multi-provider support. Konfigurasi via file <code>config.yaml</code> — tambahkan Lintasan sebagai custom provider.`,
          en_content: `Hermes Agent is an AI assistant with multi-provider support. Configure via <code>config.yaml</code> — add Lintasan as a custom provider.`,
          id_code: `# ~/.hermes/config.yaml
models:
  providers:
    lintasan:
      base_url: "http://localhost:20180/v1"
      api_key: "lintasan-master-key"
      models:
        - "gpt-4o"
        - "claude-sonnet-4"
        - "gemini-2.5-pro"

# Hermes sekarang bisa pakai semua model via Lintasan
hermes config set provider lintasan
hermes config set model gpt-4o`,
          en_code: `# ~/.hermes/config.yaml
models:
  providers:
    lintasan:
      base_url: "http://localhost:20180/v1"
      api_key: "lintasan-master-key"
      models:
        - "gpt-4o"
        - "claude-sonnet-4"
        - "gemini-2.5-pro"

# Hermes now uses all models via Lintasan
hermes config set provider lintasan
hermes config set model gpt-4o`,
          language: 'yaml',
        },
        {
          id: 'opencode',
          id_title: 'OpenCode CLI',
          en_title: 'OpenCode CLI',
          id_content: `OpenCode adalah open-source AI coding agent CLI. Ganti endpoint ke Lintasan via env vars.`,
          en_content: `OpenCode is an open-source AI coding agent CLI. Switch endpoint to Lintasan via env vars.`,
          id_code: `export OPENAI_API_KEY="lintasan-master-key"
export OPENAI_ENDPOINT="http://localhost:20180/v1"

opencode "Buat REST API untuk user management"`,
          en_code: `export OPENAI_API_KEY="lintasan-master-key"
export OPENAI_ENDPOINT="http://localhost:20180/v1"

opencode "Create a REST API for user management"`,
          language: 'bash',
        },
        {
          id: 'zed',
          id_title: 'Zed Editor',
          en_title: 'Zed Editor',
          id_content: `Zed punya fitur AI assistant built-in. Tambahkan Lintasan sebagai provider di settings.`,
          en_content: `Zed has a built-in AI assistant. Add Lintasan as a provider in settings.`,
          id_code: `// ~/.config/zed/settings.json
{
  "assistant": {
    "provider": {
      "name": "lintasan",
      "type": "open_ai_compatible",
      "api_url": "http://localhost:20180/v1",
      "available_models": [
        "gpt-4o",
        "claude-sonnet-4",
        "gemini-2.5-pro"
      ]
    },
    "default_model": "gpt-4o"
  }
}`,
          en_code: `// ~/.config/zed/settings.json
{
  "assistant": {
    "provider": {
      "name": "lintasan",
      "type": "open_ai_compatible",
      "api_url": "http://localhost:20180/v1",
      "available_models": [
        "gpt-4o",
        "claude-sonnet-4",
        "gemini-2.5-pro"
      ]
    },
    "default_model": "gpt-4o"
  }
}`,
          language: 'json',
        },
        {
          id: 'cursor',
          id_title: 'Cursor IDE',
          en_title: 'Cursor IDE',
          id_content: `Cursor IDE support OpenAI-compatible API. Bisa diarahkan ke Lintasan.`,
          en_content: `Cursor IDE supports OpenAI-compatible API. Can be pointed to Lintasan.`,
          id_code: `// Cursor Settings → Models → OpenAI API Key
// Isi dengan:
API Key: lintasan-master-key
Base URL: http://localhost:20180/v1

// Semua model di Lintasan otomatis muncul di Cursor`,
          en_code: `// Cursor Settings → Models → OpenAI API Key
// Fill in:
API Key: lintasan-master-key
Base URL: http://localhost:20180/v1

// All Lintasan models automatically appear in Cursor`,
          language: 'plaintext',
        },
        {
          id: 'continue-dev',
          id_title: 'Continue.dev (VS Code / JetBrains)',
          en_title: 'Continue.dev (VS Code / JetBrains)',
          id_content: `Continue.dev adalah AI code assistant open-source untuk VS Code dan JetBrains.`,
          en_content: `Continue.dev is an open-source AI code assistant for VS Code and JetBrains.`,
          id_code: `// ~/.continue/config.json
{
  "models": [
    {
      "title": "Lintasan GPT-4o",
      "provider": "openai",
      "model": "gpt-4o",
      "apiKey": "lintasan-master-key",
      "apiBase": "http://localhost:20180/v1"
    },
    {
      "title": "Lintasan Claude",
      "provider": "openai",
      "model": "claude-sonnet-4",
      "apiKey": "lintasan-master-key",
      "apiBase": "http://localhost:20180/v1"
    }
  ]
}`,
          en_code: `// ~/.continue/config.json
{
  "models": [
    {
      "title": "Lintasan GPT-4o",
      "provider": "openai",
      "model": "gpt-4o",
      "apiKey": "lintasan-master-key",
      "apiBase": "http://localhost:20180/v1"
    },
    {
      "title": "Lintasan Claude",
      "provider": "openai",
      "model": "claude-sonnet-4",
      "apiKey": "lintasan-master-key",
      "apiBase": "http://localhost:20180/v1"
    }
  ]
}`,
          language: 'json',
        },
        {
          id: 'aider',
          id_title: 'Aider',
          en_title: 'Aider',
          id_content: `Aider adalah AI pair programming CLI. Support OpenAI-compatible API.`,
          en_content: `Aider is an AI pair programming CLI. Supports OpenAI-compatible API.`,
          id_code: `export OPENAI_API_KEY="lintasan-master-key"
export OPENAI_API_BASE="http://localhost:20180/v1"

aider --model gpt-4o`,
          en_code: `export OPENAI_API_KEY="lintasan-master-key"
export OPENAI_API_BASE="http://localhost:20180/v1"

aider --model gpt-4o`,
          language: 'bash',
        },
        {
          id: 'openclaw',
          id_title: 'OpenClaw',
          en_title: 'OpenClaw',
          id_content: `OpenClaw adalah AI coding agent open-source.`,
          en_content: `OpenClaw is an open-source AI coding agent.`,
          id_code: `export OPENAI_API_KEY="lintasan-master-key"
export OPENAI_BASE_URL="http://localhost:20180/v1"

openclaw chat "Implement authentication middleware"`,
          en_code: `export OPENAI_API_KEY="lintasan-master-key"
export OPENAI_BASE_URL="http://localhost:20180/v1"

openclaw chat "Implement authentication middleware"`,
          language: 'bash',
        },
      ],
    },
  ];

  function toggleSection(id: string) {
    if (expandedSections.has(id)) {
      expandedSections.delete(id);
    } else {
      expandedSections.add(id);
    }
    expandedSections = new Set(expandedSections);
  }

  function scrollToSubsection(sectionId: string, subsectionId: string) {
    activeSection = sectionId;
    if (!expandedSections.has(sectionId)) {
      expandedSections.add(sectionId);
      expandedSections = new Set(expandedSections);
    }
    requestAnimationFrame(() => {
      const el = document.getElementById(`doc-${subsectionId}`);
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
  }

  async function copyCode(code: string) {
    await navigator.clipboard.writeText(code);
    copiedCode = code;
    setTimeout(() => { copiedCode = null; }, 2000);
  }

  function renderDocContent(content: string): string {
    let html = content
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');

    html = html.replace(/`([^`]+)`/g, '<code class="doc-inline-code">$1</code>');
    html = html.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
    html = html.replace(/\*([^*]+)\*/g, '<em>$1</em>');
    html = html.replace(/^(\|[^\n]+\|)$/gm, (match) => {
      return match.replace(/\|([^|]+)\|/g, '| <span class="table-cell">$1</span> |');
    });
    html = html.replace(/\n\n/g, '</p><p style="margin-top: 8px;">');
    html = html.replace(/\n/g, '<br>');

    return `<p>${html}</p>`;
  }

  let filteredDocs = $derived(
    searchQuery.trim()
      ? docs.map(section => ({
          ...section,
          subsections: section.subsections.filter(
            sub =>
              (lang === 'id' ? sub.id_title : sub.en_title).toLowerCase().includes(searchQuery.toLowerCase()) ||
              (lang === 'id' ? sub.id_content : sub.en_content).toLowerCase().includes(searchQuery.toLowerCase())
          ),
        })).filter(section => section.subsections.length > 0)
      : docs
  );

  function t(idText: string, enText: string): string {
    return lang === 'id' ? idText : enText;
  }
</script>

<svelte:head>
  <title>Docs — Lintasan</title>
</svelte:head>

<div style="display: flex; gap: 24px; min-height: calc(100vh - var(--header-h) - 48px);">
  <!-- Sidebar -->
  <div
    class="docs-sidebar"
    style="
      width: 260px; flex-shrink: 0;
      background: var(--color-bg-card);
      border: 1px solid var(--color-border);
      border-radius: var(--radius);
      overflow: hidden;
      position: sticky; top: calc(var(--header-h) + 24px);
      height: fit-content; max-height: calc(100vh - var(--header-h) - 48px);
      display: flex; flex-direction: column;
    "
  >
    <!-- Language toggle + Search -->
    <div style="padding: 14px; border-bottom: 1px solid var(--color-border); display: flex; flex-direction: column; gap: 10px;">
      <!-- Lang toggle -->
      <div style="display: flex; gap: 6px;">
        <button
          class="lang-btn"
          class:active={lang === 'id'}
          onclick={() => lang = 'id'}
        >🇮🇩 ID</button>
        <button
          class="lang-btn"
          class:active={lang === 'en'}
          onclick={() => lang = 'en'}
        >🇬🇧 EN</button>
      </div>
      <!-- Search -->
      <div style="position: relative;">
        <Search size={14} style="color: var(--color-fg-3); position: absolute; left: 10px; top: 50%; transform: translateY(-50%); pointer-events: none;" />
        <input
          class="input-field"
          placeholder={lang === 'id' ? 'Cari docs...' : 'Search docs...'}
          bind:value={searchQuery}
          style="padding-left: 32px; font-size: 12px;"
        />
      </div>
    </div>

    <!-- Navigation -->
    <nav style="flex: 1; overflow-y: auto; padding: 12px 10px;">
      {#each filteredDocs as section}
        <div style="margin-bottom: 4px;">
          <button
            class="sidebar-section-btn"
            class:active={activeSection === section.id}
            onclick={() => {
              activeSection = section.id;
              toggleSection(section.id);
            }}
          >
            <section.icon size={16} stroke-width={1.8} />
            <span style="flex: 1; text-align: left;">{t(section.id_title, section.en_title)}</span>
            {#if expandedSections.has(section.id)}
              <ChevronDown size={14} />
            {:else}
              <ChevronRight size={14} />
            {/if}
          </button>

          {#if expandedSections.has(section.id)}
            <div style="padding-left: 12px; animation: fadeInUp 0.2s ease-out;">
              {#each section.subsections as sub}
                <button
                  class="sidebar-sub-btn"
                  onclick={() => scrollToSubsection(section.id, sub.id)}
                >
                  {t(sub.id_title, sub.en_title)}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
    </nav>
  </div>

  <!-- Main content -->
  <div style="flex: 1; min-width: 0;">
    <div style="display: flex; flex-direction: column; gap: 24px;">
      {#if filteredDocs.length === 0}
        <div class="card">
          <div class="flex flex-col items-center justify-center" style="padding: 48px; opacity: 0.6;">
            <Search size={48} style="color: var(--color-fg-3); margin-bottom: 12px;" stroke-width={1.2} />
            <div style="font-size: 14px; font-weight: 500; color: var(--color-fg-2);">
              {lang === 'id' ? 'Tidak ditemukan' : 'No results found'}
            </div>
            <div style="font-size: 13px; color: var(--color-fg-3); margin-top: 4px;">
              {lang === 'id' ? 'Coba kata kunci lain' : 'Try a different search term'}
            </div>
          </div>
        </div>
      {:else}
        {#each filteredDocs as section}
          <div class="card" style="padding: 0; overflow: hidden;">
            <!-- Section header -->
            <button
              class="section-header"
              onclick={() => toggleSection(section.id)}
            >
              <div class="flex items-center gap-3">
                <div
                  class="flex items-center justify-center rounded-lg"
                  style="width: 36px; height: 36px; background: var(--color-primary-light);"
                >
                  <section.icon size={18} style="color: var(--color-primary);" />
                </div>
                <span style="font-size: 16px; font-weight: 600; color: var(--color-fg-0);">
                  {t(section.id_title, section.en_title)}
                </span>
                <span style="font-size: 12px; color: var(--color-fg-3); font-weight: 400;">
                  {section.subsections.length} {lang === 'id' ? 'bagian' : 'section'}{section.subsections.length !== 1 ? (lang === 'id' ? '' : 's') : ''}
                </span>
              </div>
              {#if expandedSections.has(section.id)}
                <ChevronDown size={18} style="color: var(--color-fg-3);" />
              {:else}
                <ChevronRight size={18} style="color: var(--color-fg-3);" />
              {/if}
            </button>

            <!-- Subsections -->
            {#if expandedSections.has(section.id)}
              <div style="border-top: 1px solid var(--color-border);">
                {#each section.subsections as sub, i}
                  <div
                    id="doc-{sub.id}"
                    class="doc-subsection"
                    style="animation: fadeInUp {0.2 + i * 0.08}s ease-out;"
                  >
                    <h3 style="font-size: 15px; font-weight: 600; color: var(--color-fg-0); margin-bottom: 12px;">
                      {t(sub.id_title, sub.en_title)}
                    </h3>

                    <div class="doc-content">
                      {@html renderDocContent(t(sub.id_content, sub.en_content))}
                    </div>

                    {#if (lang === 'id' ? sub.id_code : sub.en_code)}
                      <div class="code-container">
                        <div class="code-header">
                          <span style="font-size: 11px; font-weight: 500; color: var(--color-fg-3); text-transform: uppercase; letter-spacing: 0.5px;">
                            {sub.language || 'code'}
                          </span>
                          <button
                            class="code-copy-btn"
                            onclick={() => copyCode((lang === 'id' ? sub.id_code : sub.en_code)!)}
                            title={lang === 'id' ? 'Salin kode' : 'Copy code'}
                          >
                            {#if copiedCode === (lang === 'id' ? sub.id_code : sub.en_code)}
                              <Check size={12} style="color: var(--color-success);" />
                            {:else}
                              <Copy size={12} />
                            {/if}
                          </button>
                        </div>
                        <pre class="code-block"><code>{lang === 'id' ? sub.id_code : sub.en_code}</code></pre>
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      {/if}
    </div>
  </div>
</div>

<style>
  .docs-sidebar {
    display: block;
  }
  @media (max-width: 768px) {
    .docs-sidebar {
      display: none !important;
    }
  }

  .lang-btn {
    flex: 1;
    padding: 6px 10px;
    border: 1px solid var(--color-border);
    background: transparent;
    color: var(--color-fg-2);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    border-radius: 6px;
    transition: var(--transition);
  }
  .lang-btn:hover {
    background: var(--color-bg-body);
  }
  .lang-btn.active {
    background: var(--color-primary);
    color: #fff;
    border-color: var(--color-primary);
  }

  .sidebar-section-btn {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 10px 12px;
    border: none;
    background: transparent;
    color: var(--color-fg-1);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: var(--transition);
  }
  .sidebar-section-btn:hover {
    background: var(--color-bg-body);
    color: var(--color-fg-0);
  }
  .sidebar-section-btn.active {
    color: var(--color-primary);
    background: var(--color-primary-light);
    font-weight: 600;
  }

  .sidebar-sub-btn {
    display: block;
    width: 100%;
    padding: 7px 12px;
    border: none;
    background: transparent;
    color: var(--color-fg-2);
    font-size: 12px;
    font-weight: 400;
    cursor: pointer;
    text-align: left;
    border-radius: var(--radius-sm);
    transition: var(--transition);
  }
  .sidebar-sub-btn:hover {
    background: var(--color-bg-body);
    color: var(--color-fg-0);
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 18px 20px;
    border: none;
    background: transparent;
    cursor: pointer;
    transition: var(--transition);
  }
  .section-header:hover {
    background: var(--color-bg-body);
  }

  .doc-subsection {
    padding: 20px 24px;
    border-bottom: 1px solid var(--color-border-light);
  }
  .doc-subsection:last-child {
    border-bottom: none;
  }

  .doc-content {
    font-size: 13px;
    color: var(--color-fg-1);
    line-height: 1.7;
  }
  .doc-content :global(.doc-inline-code) {
    background: var(--color-bg-body);
    padding: 2px 6px;
    border-radius: 4px;
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--color-primary);
  }

  .code-container {
    margin-top: 14px;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    overflow: hidden;
  }
  .code-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 14px;
    background: var(--color-bg-body);
    border-bottom: 1px solid var(--color-border);
  }
  .code-copy-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px;
    border: none;
    background: transparent;
    color: var(--color-fg-3);
    cursor: pointer;
    border-radius: 4px;
    font-size: 11px;
    transition: var(--transition);
  }
  .code-copy-btn:hover {
    background: var(--color-border-light);
    color: var(--color-fg-1);
  }
  .code-block {
    padding: 16px;
    margin: 0;
    background: var(--color-bg-card);
    overflow-x: auto;
    font-family: var(--font-mono);
    font-size: 12px;
    line-height: 1.7;
    color: var(--color-fg-1);
  }
</style>
