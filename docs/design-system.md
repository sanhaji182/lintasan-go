# Lintasan Design System — SvelteKit Reference

## Color Palette

### Light Theme
- bg-body: #f1f5f9 (Slate 100)
- bg-card: #ffffff
- bg-sidebar: #ffffff
- bg-sidebar-hover: #f1f5f9
- fg-0: #1e293b (Slate 800)
- fg-1: #334155 (Slate 700)
- fg-2: #64748b (Slate 500)
- fg-3: #94a3b8 (Slate 400)
- border: #e2e8f0 (Slate 200)
- border-light: #f1f5f9
- primary: #3c50e0 (Indigo 600)
- primary-hover: #4f63e8
- primary-glow: rgba(60,80,224,0.2)
- primary-light: #eff4ff
- success: #10b981 / light: #ecfdf5
- warning: #f59e0b / light: #fffbeb
- error: #ef4444 / light: #fef2f2
- info: #3b82f6 / light: #eff6ff
- purple: #8b5cf6 / light: #f5f3ff
- sidebar-border: #e2e8f0
- logo-gradient: linear-gradient(135deg, #3c50e0 0%, #6366f1 100%)

### Dark Theme
- bg-body: #0f1419
- bg-card: #1a2332
- bg-sidebar: #111827
- bg-sidebar-hover: #1f2937
- fg-0: #f1f5f9
- fg-1: #cbd5e1
- fg-2: #94a3b8
- fg-3: #64748b
- border: #1e293b
- border-light: #1e293b
- primary-light: rgba(60,80,224,0.12)
- success-light: rgba(16,185,129,0.12)
- warning-light: rgba(245,158,11,0.12)
- error-light: rgba(239,68,68,0.12)
- info-light: rgba(59,130,246,0.12)
- purple-light: rgba(139,92,246,0.12)
- sidebar-border: #1e293b

## Typography
- Font: 'Inter', system-ui, -apple-system, sans-serif
- Mono: 'JetBrains Mono', ui-monospace, SFMono-Regular, monospace
- Sizes: 10px (sidebar groups), 11px (table headers/badges), 12px (descriptions), 13px (body/nav), 14px (card titles), 15px (page title), 18px (h1), 20px (metrics), 24px (large metrics), 28px (animated values)
- Weights: 400 (body), 500 (nav/labels), 600 (active/headings), 700 (numbers/titles)
- Line-height: 1.6

## Spacing
- 8px (card sections), 12px (form fields), 16px (card internal), 20px (card padding/grid), 24px (page content), 28px (section margins), 60px (header height)

## Shadows
- sm: 0 1px 2px rgba(0,0,0,0.04) / dark: rgba(0,0,0,0.3)
- : 0 2px 8px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04) / dark: 0 2px 8px rgba(0,0,0,0.4)
- md: 0 8px 24px rgba(0,0,0,0.08) / dark: rgba(0,0,0,0.5)
- lg: 0 16px 48px rgba(0,0,0,0.1) / dark: rgba(0,0,0,0.6)

## Border Radius
- sm: 8px (nav/inputs/buttons)
- : 12px (cards/panels)
- lg: 16px (login card)
- pill: 9999px (badges)
- 6px (code snippets, small buttons)
- 4px (selects, progress bars)

## Sidebar
- Width: 260px
- Logo: 36x36px gradient icon "L" + "Lintasan" 14px/700 + version 11px/mono
- Groups: MENU (7 items), MANAGE (6 items), TOOLS (3 items) — total 16 pages
- Active: primary color + primary-light bg + 3px left border + dot pulse
- Theme toggle: bottom of sidebar

## Animations
- Default: 0.2s cubic-bezier(0.4, 0, 0.2, 1)
- Slow: 0.4s cubic-bezier(0.4, 0, 0.2, 1)
- Spring: 0.5s cubic-bezier(0.34, 1.56, 0.64, 1)
- fadeInUp: opacity 0→1, translateY 16px→0
- shimmer: background-position -200%→200%
- dotPulse: scale 1→1.4→1
- Stagger: 60ms per child
- AnimatedNumber: 800ms ease-out cubic

## Responsive
- Single breakpoint: 768px
- Mobile: sidebar hidden (slide-in), grid 1fr, font 14px
- Touch targets: min 44px

## Pages (16)
1. Overview — metric cards (4-col grid), charts, recent requests
2. Accounts — connection cards, test/sync/delete
3. Routing — combo list, load balancer, aliases
4. Fallback — model/connection chains
5. Logs — filterable table
6. Usage — provider/model breakdown, daily chart
7. Analytics — token savings, cache perf, daily chart
8. API Keys — key list, create/copy/delete
9. Teams — team list, members
10. Users — user CRUD
11. Webhooks — webhook CRUD
12. Backup — export/import
13. Settings — toggles, inputs
14. Plugins — tabs (installed/store/generate)
15. Playground — chat interface
16. Docs — inline docs
