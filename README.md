# Melovian

> [!WARNING]
> This project is still alpha level software and being actively developed.

Melovian plays music from a library you control. Connect [Navidrome](https://www.navidrome.org/) or any Subsonic-compatible server, or add folders on your computer and play everything in one app.

<p>
  <img src="showcase/desktop-dark-home.png" alt="Home (desktop, dark)" width="480" />
</p>

More shots in [`showcase/`](showcase/).

## Install

Grab a build from the [releases page](https://github.com/melovian-hq/Melovian/releases):

- Desktop: Linux tar + AppImage, Windows zip, macOS universal
- Mobile: Android APK, ad-hoc iOS IPA (resign with your Apple ID)
- Server binaries for Linux, the BSDs, Windows, and macOS
- Docker: `docker pull ghcr.io/melovian-hq/melovian:latest`

## Build from source

You need Go 1.26+, Node.js 22+, pnpm 11, and the Wails v3 CLI. On Linux add a C compiler plus `libmpv`, GTK 4, and WebKitGTK dev packages.

```bash
cp .env.example .env
cd frontend && pnpm install && pnpm run build && cd ..
CGO_ENABLED=1 go build -tags production -trimpath -buildvcs=false -ldflags="-w -s" -o bin/melovian
```

Run the desktop app in dev mode with hot reload:

```bash
wails3 dev -config ./build/config.yml -port 9245
```

Headless server (no GUI):

```bash
go build -tags server -o bin/melovian-server
./bin/melovian-server
```

Docker (web):

```bash
docker build -t melovian:web -f docker/Dockerfile .
cd docker && docker compose --env-file ../.env up -d
```

The repo also ships a Taskfile that wraps these commands. Run `task --list` to see it.

## Docs

Guides under [`docs/en/`](docs/en/):

- [Getting started](docs/en/getting-started.md)
- [Requirements](docs/en/requirements.md)
- [Features and platforms](docs/en/features.md)
- [Build and packaging](docs/en/build.md)
- [Docker / web deploy](docs/en/docker.md)
- [Configuration](docs/en/configuration.md)
- [Development](docs/en/development.md)

Project docs at the repo root:

- [Changelog](CHANGELOG.md)
- [Contributing](CONTRIBUTING.md)
- [Security](SECURITY.md)
- [TODO](TODO.md)

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full text.
