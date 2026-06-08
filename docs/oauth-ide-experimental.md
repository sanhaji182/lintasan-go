# OAuth IDE (Experimental)

Lab-only feature. **Default OFF** (`LINTASAN_OAUTH_IDE_ENABLED` unset or `false`).

## Enable (private instance)

Set environment variables before `lintasan start`:

- `LINTASAN_OAUTH_IDE_ENABLED=true`
- `LINTASAN_OAUTH_PUBLIC_BASE_URL` — public HTTPS origin (must match OAuth app redirect)
- Per provider: `LINTASAN_OAUTH_IDE_<PROVIDER>_CLIENT_ID` and `_CLIENT_SECRET`
- Non-GitHub providers: also `LINTASAN_OAUTH_IDE_<PROVIDER>_TOKEN_URL`

Example provider key: `COPILOT`, `CURSOR`, `CLAUDE_DESKTOP` (hyphens become underscores).

## Behavior

- **Admin-only** for authorize, list sessions, revoke.
- **Public** only `GET /api/oauth/callback/{provider}` when flag is ON (validates `state` = pending session).
- **No stub tokens** — callback fails if OAuth app env is missing.
- **Proxy** does not consume tokens yet (`proxy_wired: false` in `/api/oauth/status`).

## ToS

See disclaimer in dashboard. Not a substitute for official API keys. Multi-tenant / resale is out of scope.