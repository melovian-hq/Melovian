# Features

- Browse artists, albums, genres, and search your library
- Connect multiple Subsonic servers and switch between them
- Index local music folders alongside server libraries
- Local playlists and server playlists from Navidrome
- Favorites that stay in sync with server stars
- Personalized mixes and play history
- Lyrics with configurable providers
- Optional Lyrics Whisper extension: synced lyrics generated from track audio via a whisper.cpp-compatible server or a client-side WASM engine
- Metadata editor with catalog matches from iTunes, MusicBrainz, Deezer, and TheAudioDB
- Offline track cache with a clear on-player indicator
- Crossfade, transcoding options, and keyboard shortcuts (press `?` for help)
- Command palette (`Ctrl/Cmd + K`)
- Linux desktop: MPRIS media keys, system tray, and AppImage builds with bundled libmpv
- Android: lock-screen / notification media controls, audio focus, and Android Auto (Now Playing + queue browse)
- iOS: system Now Playing and CarPlay Now Playing via remote commands (full CarPlay library browse needs Apple's carplay-audio entitlement)
- Web mode via Docker with per-user accounts, optional OIDC sign-in, and Prometheus metrics

## Platforms

| Platform | Status |
|----------|--------|
| Linux | Desktop app, AppImage, and packages |
| Windows | Desktop app |
| macOS | Desktop app |
| Android | APK (HTML audio in WebView), Android Auto media session |
| iOS | IPA / simulator (HTML audio in WKWebView), CarPlay Now Playing |
| Web (Docker) | Server mode, no native GUI |

## Mobile media notes

Android keeps a `mediaPlayback` foreground service, requests audio focus on play, and exports a `MediaBrowserService` for Android Auto. The WebView pushes track metadata and the current queue over `window.wails.setMediaState` and `setMediaQueue`. Notification taps reopen the app. `POST_NOTIFICATIONS` is requested when playback starts.

iOS updates `MPNowPlayingInfoCenter` and `MPRemoteCommandCenter` from the same frontend payload (via the `melovianMedia` script message handler). That is enough for CarPlay's Now Playing screen while audio is routed to the car. On-screen CarPlay library templates are not included in this build.
