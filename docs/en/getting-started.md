# Getting started

1. Copy the example environment file and edit it if needed (or run `task setup`):

   ```bash
   cp .env.example .env
   ```

2. Install frontend dependencies (also done by `task setup`):

   ```bash
   cd frontend && pnpm install
   ```

3. Start the app in development mode:

   ```bash
   task dev
   ```

   Server-mode UI (no Wails window): `task dev:server`, then open http://127.0.0.1:9245.

   You can also run `wails3 dev` directly for desktop.

On first launch, open **Settings → Servers** to add a Subsonic server or a local music folder. Optional `NAVIDROME_*` variables in `.env` can pre-fill your first server connection.

See [Requirements](requirements.md) for toolchain details.
