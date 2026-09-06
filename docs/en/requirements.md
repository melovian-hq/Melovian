# Requirements

## Desktop development

- [Go](https://go.dev/) 1.26 or newer (`go.mod`)
- [Node.js](https://nodejs.org/) 22+ (see `.nvmrc`) and [pnpm](https://pnpm.io/) 11 (pinned in `frontend/package.json`)
- [Wails v3 CLI](https://v3.wails.io/getting-started/installation) (version matches `go.mod`, also listed in `mise.toml`)
- [Task](https://taskfile.dev/) (optional but recommended)
- **Linux only:** a C compiler plus `libmpv`, GTK 4, and WebKitGTK development packages for native builds

Optional: [mise](https://mise.jdx.dev/) (`mise.toml`), [lefthook](https://github.com/evilmartians/lefthook), [golangci-lint](https://golangci-lint.run/). Server-mode-only work can use the Dev Container under `.devcontainer/`.

## Android / iOS

- Android SDK + NDK 26 (or set `ANDROID_NDK_HOME`)
- JDK 21 for Gradle
- Xcode 15+ for iOS
- `task toolchain:mte` builds a pinned Go tip with ARM MTE fixes. Desktop and server builds keep the Go version in `go.mod`.

## Web deployment

- Docker and Docker Compose
