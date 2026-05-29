<script lang="ts">
  import {
    BookOpen, ChevronRight, ChevronDown, Search,
    Zap, Code2, Settings, Lightbulb, ExternalLink, Copy, Check
  } from 'lucide-svelte';

  interface DocSection {
    id: string;
    title: string;
    icon: typeof BookOpen;
    subsections: DocSubsection[];
  }

  interface DocSubsection {
    id: string;
    title: string;
    content: string;
    code?: string;
    language?: string;
  }

  let activeSection = $state('getting-started');
  let searchQuery = $state('');
  let expandedSections = $state<Set<string>>(new Set(['getting-started']));
  let copiedCode = $state<string | null>(null);

  const docs: DocSection[] = [
    {
      id: 'getting-started',
      title: 'Getting Started',
      icon: Zap,
      subsections: [
        {
          id: 'quick-start',
          title: 'Quick Start',
          content: `Lintasan is an AI gateway that sits between your application and AI providers. It handles routing, caching, rate limiting, and observability automatically.

To get started, install the Lintasan CLI and initialize a new project:`,
          code: `# Install Lintasan
npm install -g @lintasan/cli

# Initialize a new project
lintasan init my-gateway

# Start the gateway
cd my-gateway && lintasan start`,
          language: 'bash',
        },
        {
          id: 'configuration-basics',
          title: 'Configuration Basics',
          content: `Configure your gateway using the \`lintasan.config.js\` file. The minimum configuration requires at least one provider connection:`,
          code: `// lintasan.config.js
export default {
  port: 20181,
  providers: [
    {
      name: 'openai',
      type: 'openai',
      apiKey: process.env.OPENAI_API_KEY,
      models: ['gpt-4o', 'gpt-4o-mini'],
    },
  ],
  cache: {
    enabled: true,
    ttl: 300, // seconds
  },
};`,
          language: 'javascript',
        },
        {
          id: 'first-request',
          title: 'Making Your First Request',
          content: `Once the gateway is running, send requests using the OpenAI-compatible API:`,
          code: `curl http://localhost:20181/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [
      { "role": "user", "content": "Hello!" }
    ]
  }'`,
          language: 'bash',
        },
      ],
    },
    {
      id: 'api-reference',
      title: 'API Reference',
      icon: Code2,
      subsections: [
        {
          id: 'chat-completions',
          title: 'Chat Completions',
          content: `The chat completions endpoint is fully compatible with the OpenAI API specification.

**Endpoint:** \`POST /v1/chat/completions\`

**Parameters:**
- \`model\` (string, required) — Model identifier
- \`messages\` (array, required) — Array of message objects
- \`temperature\` (number) — Sampling temperature (0-2)
- \`max_tokens\` (integer) — Maximum tokens to generate
- \`stream\` (boolean) — Enable streaming responses
- \`top_p\` (number) — Nucleus sampling parameter`,
          code: `{
  "model": "gpt-4o",
  "messages": [
    { "role": "system", "content": "You are helpful." },
    { "role": "user", "content": "Explain quantum computing." }
  ],
  "temperature": 0.7,
  "max_tokens": 1000,
  "stream": false
}`,
          language: 'json',
        },
        {
          id: 'embeddings',
          title: 'Embeddings',
          content: `Generate vector embeddings for text inputs.

**Endpoint:** \`POST /v1/embeddings\`

**Parameters:**
- \`model\` (string, required) — Embedding model identifier
- \`input\` (string | array, required) — Text to embed
- \`encoding_format\` (string) — Output format: "float" or "base64"`,
          code: `curl http://localhost:20181/v1/embeddings \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "text-embedding-3-small",
    "input": "The quick brown fox"
  }'`,
          language: 'bash',
        },
        {
          id: 'models-list',
          title: 'List Models',
          content: `Retrieve available models across all configured providers.

**Endpoint:** \`GET /v1/models\`

Returns a list of model objects with their capabilities and provider information.`,
          code: `curl http://localhost:20181/v1/models`,
          language: 'bash',
        },
      ],
    },
    {
      id: 'configuration',
      title: 'Configuration',
      icon: Settings,
      subsections: [
        {
          id: 'providers',
          title: 'Provider Setup',
          content: `Lintasan supports multiple AI providers. Each provider needs its own configuration block with API credentials and supported models.

**Supported providers:**
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude 3.5, Claude 3)
- Google (Gemini Pro)
- Mistral (Mixtral, Mistral)
- Custom (any OpenAI-compatible API)`,
          code: `providers: [
  {
    name: 'openai',
    type: 'openai',
    apiKey: process.env.OPENAI_API_KEY,
    models: ['gpt-4o', 'gpt-4o-mini'],
  },
  {
    name: 'anthropic',
    type: 'anthropic',
    apiKey: process.env.ANTHROPIC_API_KEY,
    models: ['claude-3.5-sonnet', 'claude-3-haiku'],
  },
  {
    name: 'custom',
    type: 'openai-compatible',
    baseUrl: 'https://my-api.example.com/v1',
    apiKey: process.env.CUSTOM_API_KEY,
    models: ['custom-model-1'],
  },
]`,
          language: 'javascript',
        },
        {
          id: 'routing-rules',
          title: 'Routing Rules',
          content: `Configure intelligent routing to direct requests to specific providers based on model, cost, latency, or custom rules.`,
          code: `routing: {
  strategy: 'cost-optimized', // or 'latency', 'round-robin'
  rules: [
    {
      match: { model: 'gpt-4*' },
      targets: ['openai', 'azure-openai'],
    },
    {
      match: { model: 'claude-*' },
      targets: ['anthropic'],
    },
  ],
  fallback: {
    enabled: true,
    maxRetries: 3,
  },
}`,
          language: 'javascript',
        },
        {
          id: 'caching',
          title: 'Caching',
          content: `Enable semantic caching to reduce costs and improve latency for repeated or similar requests.`,
          code: `cache: {
  enabled: true,
  ttl: 300,
  strategy: 'semantic',  // or 'exact'
  similarity: 0.95,       // threshold for semantic match
  excludeModels: ['gpt-4-turbo'], // models to skip caching
}`,
          language: 'javascript',
        },
      ],
    },
    {
      id: 'examples',
      title: 'Examples',
      icon: Lightbulb,
      subsections: [
        {
          id: 'nodejs-client',
          title: 'Node.js Client',
          content: `Use the OpenAI SDK pointed at your Lintasan gateway:`,
          code: `import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:20181/v1',
  apiKey: 'your-lintasan-api-key',
});

const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [
    { role: 'user', content: 'Hello!' },
  ],
});

console.log(response.choices[0].message.content);`,
          language: 'typescript',
        },
        {
          id: 'streaming-example',
          title: 'Streaming Responses',
          content: `Stream responses for real-time output:`,
          code: `const stream = await client.chat.completions.create({
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
          title: 'Python Client',
          content: `Use the OpenAI Python library with Lintasan:`,
          code: `from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:20181/v1",
    api_key="your-lintasan-api-key",
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "user", "content": "Hello!"}
    ],
)

print(response.choices[0].message.content)`,
          language: 'python',
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

    // Inline code
    html = html.replace(/`([^`]+)`/g, '<code class="doc-inline-code">$1</code>');
    // Bold
    html = html.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
    // Italic
    html = html.replace(/\*([^*]+)\*/g, '<em>$1</em>');
    // Line breaks (double newline = paragraph break)
    html = html.replace(/\n\n/g, '</p><p style="margin-top: 8px;">');
    // Single line breaks
    html = html.replace(/\n/g, '<br>');

    return `<p>${html}</p>`;
  }

  let filteredDocs = $derived(
    searchQuery.trim()
      ? docs.map(section => ({
          ...section,
          subsections: section.subsections.filter(
            sub =>
              sub.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
              sub.content.toLowerCase().includes(searchQuery.toLowerCase())
          ),
        })).filter(section => section.subsections.length > 0)
      : docs
  );
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
    <!-- Search -->
    <div style="padding: 14px; border-bottom: 1px solid var(--color-border);">
      <div style="position: relative;">
        <Search size={14} style="color: var(--color-fg-3); position: absolute; left: 10px; top: 50%; transform: translateY(-50%); pointer-events: none;" />
        <input
          class="input-field"
          placeholder="Search docs..."
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
            <span style="flex: 1; text-align: left;">{section.title}</span>
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
                  {sub.title}
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
            <div style="font-size: 14px; font-weight: 500; color: var(--color-fg-2);">No results found</div>
            <div style="font-size: 13px; color: var(--color-fg-3); margin-top: 4px;">Try a different search term</div>
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
                  {section.title}
                </span>
                <span style="font-size: 12px; color: var(--color-fg-3); font-weight: 400;">
                  {section.subsections.length} section{section.subsections.length !== 1 ? 's' : ''}
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
                      {sub.title}
                    </h3>

                    <div class="doc-content">
                      {@html renderDocContent(sub.content)}
                    </div>

                    {#if sub.code}
                      <div class="code-container">
                        <div class="code-header">
                          <span style="font-size: 11px; font-weight: 500; color: var(--color-fg-3); text-transform: uppercase; letter-spacing: 0.5px;">
                            {sub.language || 'code'}
                          </span>
                          <button
                            class="code-copy-btn"
                            onclick={() => copyCode(sub.code!)}
                            title="Copy code"
                          >
                            {#if copiedCode === sub.code}
                              <Check size={12} style="color: var(--color-success);" />
                            {:else}
                              <Copy size={12} />
                            {/if}
                          </button>
                        </div>
                        <pre class="code-block"><code>{sub.code}</code></pre>
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
