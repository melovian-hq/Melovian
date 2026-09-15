# Build

Build a desktop binary for your current operating system:

```bash
cd frontend && pnpm build && cd ..
CGO_ENABLED=1 go build -tags production -trimpath -buildvcs=false -ldflags="-w -s" -o bin/melovian
```

The packaged pipeline (per-OS installers and archives) is Taskfile-orchestrated:

```bash
task build      # same result via the OS-specific task
task package    # AppImage, deb, rpm, AUR on Linux; equivalents elsewhere
```

Linux AppImage with bundled libmpv:

```bash
task package:appimage
```

Server mode (HTTP API only, no GUI window):

```bash
cd frontend && pnpm build && cd ..
go build -tags server -o bin/melovian-server .
./bin/melovian-server
```

`task build:server` adds version stamping (`-X melovian/internal/compat.Version=...` and the update public key) on top of that.

Cross-compile server binaries for release OS/arch targets (CGO off):

```bash
VERSION=v0.1.0 bash build/scripts/cross-server.sh
```

Zip the production SPA (`melovian-<version>-frontend.zip` under `bin/`):

```bash
cd frontend && pnpm build && cd dist && zip -qr ../../bin/melovian-$(git describe --tags --always --dirty)-frontend.zip . && cd ../..
```

Static frontend demo (in-browser dummy catalog, no server process):

```bash
go run ./internal/democatalog/cmd/export -out frontend/public/demo
cd frontend && VITE_STATIC_DEMO=true pnpm build
cp dist/index.html dist/404.html   # SPA fallback for static hosts
```

## Android

Needs the Android SDK/NDK and a one-time MTE Go toolchain under `build/tools/go-mte`:

```bash
bash build/scripts/ensure-go-mte.sh
task build:android
```

`task build:android:emu` builds for the host emulator ABI. `task package:android` makes a production arm64 APK. `task android:package:fat` makes a universal APK (`arm64-v8a` + `armeabi-v7a`). The Gradle/Wails pipeline is Taskfile-only.

Release APKs run R8/ProGuard. Keep rules for the Wails bridge, MediaSession, and Android Auto `MediaBrowserService` live in `build/android/app/proguard-rules.pro`.

Manual smoke for media: start a track, confirm the notification opens the app, pause from the shade, and (if you have the Desktop Head Unit) check Android Auto Now Playing and Queue.

## iOS

Requires macOS with Xcode. Simulator bundle:

```bash
bash build/scripts/ensure-go-mte.sh
task ios:package
```

Device IPA (replace the identity with yours, or `-` for ad-hoc):

```bash
task ios:package:ipa IOS_PLATFORM=device CODESIGN_IDENTITY="Apple Development: ..."
```

`Info.plist` already sets `UIBackgroundModes: audio`. Now Playing / CarPlay Now Playing wiring is in `build/ios/MelovianNowPlaying.m` and a Wails webview overlay patch (`build/ios/scripts/patch-nowplaying-overlay.py`). Full CarPlay browse UI needs the `com.apple.developer.carplay-audio` entitlement and is not built here.
