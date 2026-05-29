# 🇮🇩 Lintasan — Design System (SvelteKit Reference)

> Panduan desain untuk dashboard SvelteKit. Semua nilai menggunakan CSS custom properties — support dark/light mode otomatis.

---

# 🇬🇧 Lintasan — Design System (SvelteKit Reference)

> Design guide for the SvelteKit dashboard. All values use CSS custom properties — supports automatic dark/light mode.

---

## 🎨 Color Palette

### Light Theme ☀️

| Token | Value | Usage |
|-------|-------|-------|
| `--bg-body` | `#f1f5f9` (Slate 100) | Page background |
| `--bg-card` | `#ffffff` | Card/panel background |
| `--bg-sidebar` | `#ffffff` | Sidebar background |
| `--bg-sidebar-hover` | `#f1f5f9` | Sidebar item hover |
| `--fg-0` | `#1e293b` (Slate 800) | Primary text |
| `--fg-1` | `#334155` (Slate 700) | Secondary text |
| `--fg-2` | `#64748b` (Slate 500) | Muted text |
| `--fg-3` | `#94a3b8` (Slate 400) | Disabled/hint text |
| `--border` | `#e2e8f0` (Slate 200) | Card/input borders |
| `--border-light` | `#f1f5f9` | Subtle borders |
| `--primary` | `#3c50e0` (Indigo 600) | Primary actions, active nav |
| `--primary-hover` | `#4f63e8` | Button hover |
| `--primary-glow` | `rgba(60,80,224,0.2)` | Shadows, effects |
| `--primary-light` | `#eff4ff` | Selected/active bg |
| `--success` | `#10b981` | Success badges |
| `--success-light` | `#ecfdf5` | Success bg |
| `--warning` | `#f59e0b` | Warning badges |
| `--warning-light` | `#fffbeb` | Warning bg |
| `--error` | `#ef4444` | Error badges, delete buttons |
| `--error-light` | `#fef2f2` | Error bg |
| `--info` | `#3b82f6` | Info badges |
| `--info-light` | `#eff6ff` | Info bg |
| `--purple` | `#8b5cf6` | Special accents |
| `--purple-light` | `#f5f3ff` | Special accent bg |
| `--sidebar-border` | `#e2e8f0` | Sidebar divider |
| `--logo-gradient` | `linear-gradient(135deg, #3c50e0 0%, #6366f1 100%)` | Logo bg |

### Dark Theme 🌙

| Token | Value | Usage |
|-------|-------|-------|
| `--bg-body` | `#0f1419` | Page background |
| `--bg-card` | `#1a2332` | Card/panel background |
| `--bg-sidebar` | `#111827` | Sidebar background |
| `--bg-sidebar-hover` | `#1f2937` | Sidebar item hover |
| `--fg-0` | `#f1f5f9` | Primary text |
| `--fg-1` | `#cbd5e1` | Secondary text |
| `--fg-2` | `#94a3b8` | Muted text |
| `--fg-3` | `#64748b` | Disabled/hint text |
| `--border` | `#1e293b` | Card/input borders |
| `--primary-light` | `rgba(60,80,224,0.12)` | Selected/active bg |
| `--success-light` | `rgba(16,185,129,0.12)` | Success bg |
| `--warning-light` | `rgba(245,158,11,0.12)` | Warning bg |
| `--error-light` | `rgba(239,68,68,0.12)` | Error bg |
| `--info-light` | `rgba(59,130,246,0.12)` | Info bg |
| `--purple-light` | `rgba(139,92,246,0.12)` | Special accent bg |
| `--sidebar-border` | `#1e293b` | Sidebar divider |

---

## 🔤 Typography

| Token | Value |
|-------|-------|
| **Body font** | `'Inter', system-ui, -apple-system, sans-serif` |
| **Mono font** | `'JetBrains Mono', ui-monospace, SFMono-Regular, monospace` |
| **Base size** | 14px |
| **Line height** | 1.6 |

### Size Scale

| Size | Usage |
|------|-------|
| `10px` | Sidebar group headers, tiny badges |
| `11px` | Table headers, badges, uppercase labels |
| `12px` | Descriptions, subtitles, form hints |
| `13px` | Body text, nav items, buttons |
| `14px` | Card titles, page subtitle |
| `18px` | Page headings (h1) |
| `20px` | Metric icons |
| `22px` | Metric stat cards |
| `24px` | Login heading |
| `28px` | Animated metric values |

### Weights

| Weight | Usage |
|--------|-------|
| `400` | Body text |
| `500` | Nav items, labels, buttons |
| `600` | Active nav, card titles, headings |
| `700` | Metric numbers, page title, logo |

---

## 📐 Spacing

| Size | Usage |
|------|-------|
| `4px` | Icon gaps, tight spacing |
| `6px` | Code snippet padding |
| `8px` | Card sections, nav gaps |
| `10px` | Button inner gaps |
| `12px` | Form fields, small padding |
| `14px` | Table cell padding |
| `16px` | Card internal padding |
| `20px` | Card padding, grid gaps |
| `24px` | Page content padding |
| `28px` | Section margins |
| `60px` | Header height |

---

## 🌑 Shadows

| Token | Light | Dark |
|-------|-------|------|
| `--shadow-sm` | `0 1px 2px rgba(0,0,0,0.04)` | `0 1px 2px rgba(0,0,0,0.3)` |
| `--shadow` | `0 2px 8px rgba(0,0,0,0.06)` | `0 2px 8px rgba(0,0,0,0.4)` |
| `--shadow-md` | `0 8px 24px rgba(0,0,0,0.08)` | `0 8px 24px rgba(0,0,0,0.5)` |
| `--shadow-lg` | `0 16px 48px rgba(0,0,0,0.1)` | `0 16px 48px rgba(0,0,0,0.6)` |

---

## 🔲 Border Radius

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | `8px` | Nav items, inputs, buttons |
| `--radius` | `12px` | Cards, panels |
| `--radius-lg` | `16px` | Login card, modals |
| `pill` | `9999px` | Badges, status indicators |
| `6px` | | Code snippets, small buttons |
| `4px` | | Selects, progress bars |
| `10px` | | Logo icon |

---

## 📋 Sidebar

| Property | Value |
|----------|-------|
| **Width** | `260px` |
| **Logo** | 36×36px gradient circle with "L" + "Lintasan" 14px/700 |
| **Version** | 11px mono, below logo |
| **Groups** | 3 groups: MENU (7), MANAGE (6), TOOLS (4) — total 17 pages |
| **Active state** | Primary color text + primary-light bg + 1px left border accent |
| **Hover state** | Sidebar-hover bg, smooth transition |
| **Theme toggle** | Bottom of sidebar, full-width |

### Navigation Groups

```
MENU
  ▦ Overview        🔗 Accounts        ⇄ Routing
  ↻ Fallback        ▤ Logs            ▥ Usage
  ↗ Analytics

MANAGE
  ⚿ API Keys       👥 Teams          👤 Users
  ⌁ Webhooks        ⇩ Backup         ⚙ Settings

TOOLS
  ▣ Plugins         ◌ Playground      ◇ Models
  ▱ Docs
```

---

## ✨ Animations

| Name | Duration | Easing | Effect |
|------|----------|--------|--------|
| **Default** | 0.2s | `cubic-bezier(0.4, 0, 0.2, 1)` | Hover, focus |
| **Slow** | 0.4s | `cubic-bezier(0.4, 0, 0.2, 1)` | Page transitions |
| **Spring** | 0.5s | `cubic-bezier(0.34, 1.56, 0.64, 1)` | Bouncy reveals |
| **fadeInUp** | 0.5s | ease-out | Cards appearing |
| **fadeInScale** | 0.25s | ease-out | Modals |
| **shimmer** | 1.5s | linear | Skeleton loading |
| **dotPulse** | 1.5s | ease-in-out | Active indicator |
| **barGrow** | 0.6s | ease-out | Chart bars |
| **countUp** | 0.8s | ease-out cubic | Animated numbers |

### Stagger Delay
```
1st child: 0ms
2nd child: 60ms
3rd child: 120ms
4th child: 180ms
5th child: 240ms
→ +60ms per child
```

---

## 📱 Responsive

| Breakpoint | Behavior |
|------------|----------|
| **> 768px** | Desktop: sidebar visible, 4-col metric grids |
| **≤ 768px** | Mobile: sidebar hidden (slide-in), 1-col grids, 14px font |
| **≤ 640px** | Metric grid collapses to 1-col |
| **Touch** | Min 44px touch targets |

---

## 📄 Pages (17)

| # | Page | Key Elements |
|---|------|-------------|
| 1 | **Overview** | 4-col metric cards + charts + recent requests |
| 2 | **Accounts** | Connection cards, test/sync/delete |
| 3 | **Routing** | Combo list, load balancer config, aliases |
| 4 | **Fallback** | Model/connection fallback chains |
| 5 | **Logs** | Filterable, searchable table |
| 6 | **Usage** | Provider/model breakdown, daily chart |
| 7 | **Analytics** | Token savings, cache performance, daily chart |
| 8 | **API Keys** | Key list, create/copy/delete |
| 9 | **Teams** | Team list, member management |
| 10 | **Users** | User CRUD, role assignment |
| 11 | **Webhooks** | Webhook CRUD, event testing |
| 12 | **Backup** | Export/import, file management |
| 13 | **Settings** | Toggles, inputs, save |
| 14 | **Plugins** | Tabs: installed, store, AI generator |
| 15 | **Playground** | Chat UI, model select, streaming |
| 16 | **Models** | Catalog with pricing, sync |
| 17 | **Docs** | Inline API documentation |
