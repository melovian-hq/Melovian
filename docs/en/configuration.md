# Configuration

Environment variables can live in `.env` for local development or in your shell / compose file for deployment.

## Core

| Variable | Default | Description |
|----------|---------|-------------|
| `MELOVIAN_DATA` | XDG data directory | Where Melovian stores its database and cache |
| `MELOVIAN_DATABASE_URL` | (unset) | Optional database URL. Unset uses SQLite at `{MELOVIAN_DATA}/melovian.db`. Set `postgres://` or `postgresql://` to use PostgreSQL |
| `MELOVIAN_LISTEN` | `127.0.0.1:17337` | API listen address |
| `MELOVIAN_AUTH_SECRET` | (unset) | Enables web authentication. Required for Docker |
| `MELOVIAN_PUBLIC_URL` | (unset) | Public URL for streams and OAuth behind a reverse proxy |
| `MELOVIAN_CORS_ORIGINS` | (unset) | Extra CORS origins for credentialed clients. `https://wails.localhost` is always allowed so the Android/iOS app can use this server over http or https |
| `NAVIDROME_SERVER` | (unset) | Optional bootstrap server URL (migrated into the first saved instance) |
| `NAVIDROME_USER` | (unset) | Optional bootstrap username |
| `NAVIDROME_PASSWORD` | (unset) | Optional bootstrap password |
| `MELOVIAN_INSTANCE_KEY` | (unset) | 64-char hex key used to encrypt stored instance passwords. Defaults to a generated `instance.key` file in `MELOVIAN_DATA` |

## Demo mode and access control

| Variable | Default | Description |
|----------|---------|-------------|
| `MELOVIAN_DEMO_MODE` | (unset) | Read-only public server mode. Skips account login and blocks mutating API requests. With no `NAVIDROME_*`, serves a built-in fake catalog. With `NAVIDROME_*`, proxies that Subsonic server |
| `MELOVIAN_ALLOWED_IPS` | (unset) | Comma-separated IPs and CIDRs allowed to access the server. Empty means allow all |
| `MELOVIAN_TRUST_PROXY` | (unset) | When true, resolve client IP from `X-Real-IP` or `X-Forwarded-For` before applying the allowlist |

## Local libraries

| Variable | Default | Description |
|----------|---------|-------------|
| `MELOVIAN_LOCAL_LIBRARY` | `true` on desktop | Enable or disable local folder indexing |
| `MELOVIAN_LOCAL_LIBRARY_PATH` | (unset) | Shared music folder for multi-user deployments. Disables per-user custom paths when auth is on |
| `MELOVIAN_LOCAL_LIBRARY_ROOTS` | (unset) | Extra `:`-separated roots allowed for browsed or user-supplied library paths. On a multi-user server, custom library paths must stay under home, `/media`, `/mnt`, `/run/media`, the data directory, or these roots |

Local libraries are scanned into the configured database (SQLite by default) and show up in the same browse, search, queue, and playback UI as Subsonic sources.

## OIDC / OAuth

Requires `MELOVIAN_AUTH_SECRET`. When OIDC is configured, the first sign-in creates the account from your identity provider. Local account setup is turned off while OIDC is active.

| Variable | Default | Description |
|----------|---------|-------------|
| `MELOVIAN_OIDC_ISSUER` | (unset) | OIDC issuer URL |
| `MELOVIAN_OIDC_CLIENT_ID` | (unset) | OAuth client ID |
| `MELOVIAN_OIDC_CLIENT_SECRET` | (unset) | OAuth client secret |
| `MELOVIAN_OIDC_REDIRECT_URL` | `{MELOVIAN_PUBLIC_URL}/api/auth/oidc/callback` | Redirect URI registered with your IdP |
| `MELOVIAN_OIDC_SCOPES` | `openid profile email` | Space-separated OIDC scopes |

Sign-in: `GET /api/auth/oidc/login`
Callback: `GET /api/auth/oidc/callback`

## Connection tuning

| Variable | Default | Description |
|----------|---------|-------------|
| `MELOVIAN_CONN_MIN_DELAY_MS` | `2000` | Minimum reconnect delay |
| `MELOVIAN_CONN_MAX_DELAY_MS` | `120000` | Maximum reconnect delay |
| `MELOVIAN_CONN_BACKOFF_MULTIPLIER` | `1.6` | Reconnect backoff multiplier |
| `MELOVIAN_CONN_HEALTH_CHECK_MS` | `60000` | Health check interval |
| `MELOVIAN_CONN_OFFLINE_POLL_MS` | `5000` | Offline poll interval |
| `MELOVIAN_CONN_SELF_HEAL_MS` | `300000` | Self-heal refresh interval |
| `MELOVIAN_CONN_MAX_HISTORY` | `32` | Connection event history limit |

## Metrics

Prometheus metrics are available at `GET /metrics` on the API listener.

## Error tracking (Sentry / GlitchTip)

Melovian uses the Sentry protocol. GlitchTip and other Sentry-compatible backends work with the same DSN format.

Configure in **Settings → General → Error tracking**, or with environment variables. Env vars override stored settings when set.

| Variable | Default | Description |
|----------|---------|-------------|
| `MELOVIAN_SENTRY_DSN` | (unset) | Backend error reporting DSN. Also accepts `SENTRY_DSN` |
| `MELOVIAN_SENTRY_FRONTEND_DSN` | backend DSN | Browser DSN exposed via `GET /api/config`. Also accepts `SENTRY_FRONTEND_DSN` |
| `MELOVIAN_SENTRY_ENVIRONMENT` | (unset) | Environment tag. Also accepts `SENTRY_ENVIRONMENT` |
| `MELOVIAN_SENTRY_RELEASE` | `melovian@0.1.0` | Release tag. Also accepts `SENTRY_RELEASE` |
| `MELOVIAN_SENTRY_TRACES_SAMPLE_RATE` | `0` | Performance trace sample rate (`0` to `1`). Also accepts `SENTRY_TRACES_SAMPLE_RATE` |

When a backend DSN is set, Go panics, HTTP 5xx responses, and frontend errors received on `/api/client-log` are forwarded to your tracker. The frontend initializes `@sentry/svelte` when server and per-device client reporting are enabled.

## Desktop logging and stability (Linux)

Desktop graphics workarounds are configured in Settings → General → Graphics stability on the Linux desktop app. They are stored in `{data}/graphics-settings.json` and apply on the next launch.

| Variable | Read by | Description |
|----------|---------|-------------|
| `WEBKIT_DISABLE_DMABUF_RENDERER` | WebKitGTK | Disables the DMA-BUF/GBM renderer (default on for Melovian Linux builds) |
| `WEBKIT_DISABLE_COMPOSITING_MODE` | WebKitGTK | Disables accelerated compositing (optional, off by default) |
| `__NV_DISABLE_EXPLICIT_SYNC` | NVIDIA egl-wayland | Disables explicit sync on Wayland + NVIDIA (auto by default) |

| Variable | Default | Description |
|----------|---------|-------------|
| `MELOVIAN_LOG_LEVEL` | `info` | Go log level: `debug`, `info`, `warn`, `error` |
| `MELOVIAN_LOG_FILE` | `{data}/logs/melovian.log` | Main application log file |
| `MELOVIAN_DEBUG_PPROF` | (unset) | Enable `/debug/pprof` for heap and CPU profiling. Also exposes richer memory notes via `GET /api/debug/memory` |

Desktop logs live under your data directory (default `~/.local/share/melovian/logs/`):

| File | Contents |
|------|----------|
| `melovian.log` | Go backend: API requests, startup, shutdown |
| `client.log` | Frontend messages shipped to `/api/client-log` |
| `crashes/crash-*.log` | Go panics, HTTP handler panics, frontend errors with stack traces |

WebKitGTK subprocess crashes (GPU/GBM segfaults) kill the whole app without writing to Melovian logs. Check `coredumpctl` / `journalctl` for `WebKitWebProcess`. For stubborn driver issues, try `LIBGL_ALWAYS_SOFTWARE=1` before launching.
