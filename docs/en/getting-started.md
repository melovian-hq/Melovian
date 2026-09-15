# Getting started

1. Copy the example environment file and edit it if needed:

   ```bash
   cp .env.example .env
   ```

2. Install frontend dependencies:

   ```bash
   cd frontend && pnpm install
   ```

   Steps 1 and 2 together are `task setup` (or `bash build/scripts/setup.sh`).

3. Start the app in development mode:

   ```bash
   wails3 dev -config ./build/config.yml -port 9245
   ```

   Server-mode UI (no Wails window): `bash build/scripts/dev-server.sh`, then open http://127.0.0.1:9245.

On first launch, open **Settings → Servers** to add a Subsonic server or a local music folder. Optional `NAVIDROME_*` variables in `.env` can pre-fill your first server connection.

See [Requirements](requirements.md) for toolchain details.
