# Build

Build a desktop binary for your current operating system:

```bash
task build
```

Package a release (platform-specific installer or archive):

```bash
task package
```

Linux AppImage with bundled libmpv:

```bash
task package:appimage
```

Server mode (HTTP API only, no GUI window):

```bash
task build:server
task run:server
```

Cross-compile server binaries for release OS/arch targets (CGO off):

```bash
VERSION=v0.1.0 task build:server:cross
```

Zip the production SPA:

```bash
VERSION=v0.1.0 task package:frontend
```

Static frontend demo (in-browser dummy catalog, no server process):

```bash
task build:frontend:demo
```

## Android

Needs the Android SDK/NDK and a one-time MTE Go toolchain under `build/tools/go-mte`:

```bash
task toolchain:mte
task build:android
```

`task build:android:emu` builds for the host emulator ABI. `task package:android` makes a production arm64 APK. `task android:package:fat` makes a universal APK (`arm64-v8a` + `armeabi-v7a`).

Release APKs run R8/ProGuard. Keep rules for the Wails bridge, MediaSession, and Android Auto `MediaBrowserService` live in `build/android/app/proguard-rules.pro`.

Manual smoke for media: start a track, confirm the notification opens the app, pause from the shade, and (if you have the Desktop Head Unit) check Android Auto Now Playing and Queue.

## iOS

Requires macOS with Xcode. Simulator bundle:

```bash
task toolchain:mte
task ios:package
```

Device IPA (replace the identity with yours, or `-` for ad-hoc):

```bash
task ios:package:ipa IOS_PLATFORM=device CODESIGN_IDENTITY="Apple Development: ..."
```

`Info.plist` already sets `UIBackgroundModes: audio`. Now Playing / CarPlay Now Playing wiring is in `build/ios/MelovianNowPlaying.m` and a Wails webview overlay patch (`build/ios/scripts/patch-nowplaying-overlay.py`). Full CarPlay browse UI needs the `com.apple.developer.carplay-audio` entitlement and is not built here.
