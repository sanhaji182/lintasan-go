// Root layout config — SPA mode for embedding into the Go binary.
// The whole dashboard is client-rendered (auth guard runs in the browser via
// /api/auth/me), so we disable SSR and emit a static SPA shell that the Go
// backend serves with an index.html fallback for client-side routing.
export const ssr = false;
export const prerender = false;
export const trailingSlash = 'never';
