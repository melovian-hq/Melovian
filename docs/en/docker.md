# Web deployment (Docker)

1. Copy and edit the environment file if you have not already:

   ```bash
   cp .env.example .env
   ```

   Set `MELOVIAN_AUTH_SECRET` in `.env` to a long random string (at least 32 characters), or set `MELOVIAN_DEMO_MODE=true` for a read-only public demo. With no `NAVIDROME_*` values, demo mode serves a built-in fake catalog. Set `NAVIDROME_SERVER`, `NAVIDROME_USER`, and `NAVIDROME_PASSWORD` only when you want the demo backed by a real Subsonic server.

2. Build the frontend and Docker image, then start the stack:

   ```bash
   task build:docker
   cd docker && docker compose up -d
   ```

   Or pull a CI image from GHCR and point compose at it:

   ```bash
   docker pull ghcr.io/<owner>/melovian:latest
   # in compose, set image: ghcr.io/<owner>/melovian:latest and skip local build
   cd docker && docker compose up -d
   ```

   Tags published by `.github/workflows/docker.yml`: `latest` (master), `nightly` (schedule), semver tags (`v*`), and short `sha-*` digests. Images are multi-arch (`linux/amd64`, `linux/arm64`) with provenance attestations.

3. Open the app in your browser. Create an admin account at `/account/login`. Add a Subsonic server later from Settings or the source menu.

Stop the stack:

```bash
task stop:docker
```

The compose file uses a named volume for data, a dedicated network, a non-root container user, and a read-only root filesystem. When auth is enabled, each account keeps its own saved servers, playlists, favorites, and play history.

Set `MELOVIAN_PUBLIC_URL` when Melovian sits behind a reverse proxy so media streams and OAuth redirects use the correct address.

The Android or iOS app can use this host as its backend. In the app open Settings → Sources → Melovian server and enter your URL (`http://` on Tailscale/Netbird/LAN, or `https://` on the public internet). The server already allows the mobile origin `https://wails.localhost`. No-auth and demo servers work. Account login cookies from the mobile app need HTTPS.

## Demo deployment

For a public read-only instance without account login. Leave `NAVIDROME_*` unset to use the built-in fake catalog (real chart artist and album names, silent streams, covers fetched via iTunes lookup). Screenshots in [`showcase/`](../../showcase/) were captured this way.

```bash
MELOVIAN_DEMO_MODE=true
```

Or force the fake catalog explicitly:

```bash
MELOVIAN_DEMO_MODE=true
NAVIDROME_SERVER=fake://melovian-demo
NAVIDROME_USER=demo
NAVIDROME_PASSWORD=demo
```

To back demo mode with a real Subsonic server instead:

```bash
MELOVIAN_DEMO_MODE=true
NAVIDROME_SERVER=https://music.example.com
NAVIDROME_USER=demo
NAVIDROME_PASSWORD=your-demo-password
```

### Frontend-only static demo (GitHub Pages)

For a demo that does not run `melovian-server` at all, build the SPA with the in-browser catalog:

```bash
task demo:export
VITE_STATIC_DEMO=true VITE_BASE=/melovian/ pnpm --dir frontend run build
```

Or `task build:frontend:demo`. CI workflow `.github/workflows/pages.yml` does this on `main` when Pages is enabled for the repo (GitHub Actions source). Mutations stay blocked the same way as server demo mode.

Optional LAN restriction:

```bash
MELOVIAN_ALLOWED_IPS=192.168.0.0/16,10.0.0.0/8,127.0.0.1/32
MELOVIAN_TRUST_PROXY=true
```

Include `127.0.0.1/32` when container health checks hit localhost.

## Showcase screenshots

Capture a small set of desktop and mobile shots:

```bash
task showcase
```

Images land in [`showcase/`](../../showcase/) (six PNGs: home, album, and playlists across light/dark and desktop/mobile). The tool starts a temporary demo server with the built-in catalog.

Pass `?theme=light` or `?theme=dark` in the URL to force theme for captures.
