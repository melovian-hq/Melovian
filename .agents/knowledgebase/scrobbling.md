# Scrobbling

Melovian submits listens to Last.fm, ListenBrainz, and Rocksky. Submissions are best-effort: the frontend calls REST endpoints on the Melovian API, and the backend forwards to each service. Failures are swallowed so playback is never interrupted.

## Call flow

- `frontend/src/lib/config/music/track-boundary-ops.ts` triggers now-playing on track start and scrobbles on listen completion, alongside the upstream Subsonic `scrobble` call.
- Frontend helpers: `musicApi.rockskyNowPlaying/rockskyScrobble` and `musicApi.lastfmNowPlaying/lastfmScrobble` in `frontend/src/lib/music/api.ts`. Both take a `RockskyTrack` built by `toRockskyTrack` in `frontend/src/lib/music/rocksky.ts`.
- API paths: `/api/music/{rocksky,lastfm,listenbrainz}/{now-playing,scrobble}` plus `/api/music/settings/{rocksky,lastfm,listenbrainz}` and `.../test` (see `frontend/src/lib/core/http/api-paths.ts`).
- Backend handlers: `internal/api/lastfm.go`, `internal/api/rocksky.go`, `internal/api/listenbrainz.go`, shared track plumbing in `internal/api/scrobblers.go`. Settings UI: `SettingsLastFMPanel.svelte`, `SettingsRockskyPanel.svelte`, `SettingsListenBrainzPanel.svelte` under `frontend/src/lib/components/settings/panels/`.

## ListenBrainz

`internal/api/listenbrainz.go` posts to a configurable endpoint (default `https://api.listenbrainz.org`, so compatible instances like Maloja work too).

- Settings live in the preferences store under `store.PrefKeyListenBrainzSettings`.
- The feature is gated on the bundled `listenbrainz` extension being enabled: `extensions.IsEnabled(dataDir, "listenbrainz")`.
- `handleListenBrainzNowPlaying` and `handleListenBrainzScrobble` share `doListenBrainzScrobble`, which maps to `playing_now` / `single` listen types.
- `handleTestListenBrainzToken` validates a user token for the settings panel.

## Last.fm client

`internal/lastfm/client.go` posts to `https://ws.audioscrobbler.com/2.0/` (Audioscrobbler API 2.0).

- `UpdateNowPlaying(ctx, apiKey, apiSecret, sessionKey, track)` calls `track.updateNowPlaying`.
- `Scrobble(ctx, apiKey, apiSecret, sessionKey, track, listenedAt)` calls `track.scrobble` with a Unix timestamp.
- Requests are form-encoded and signed with the MD5 method signature (`sign` sorts params, appends the secret, hashes).

## Rocksky client

`internal/rocksky/client.go` posts to `https://audioscrobbler.rocksky.app/1/submit-listens`.

- `SubmitNowPlaying(ctx, token, track)` sends `listen_type: "playing_now"`.
- `SubmitScrobble(ctx, token, track, listenedAt)` sends `listen_type: "single"` with a Unix timestamp.
- `ValidateToken(ctx, token)` checks a token against `/1/validate-token`.

Requests use `Authorization: Token <token>` and a JSON body.

### Submission payload fields

- `listen_type`: `playing_now` or `single`
- `payload[0].listened_at`: Unix timestamp for `single`
- `payload[0].track_metadata.artist_name`
- `payload[0].track_metadata.track_name`
- `payload[0].track_metadata.release_name`
- `payload[0].track_metadata.additional_info.duration_ms`: `track.DurationSeconds * 1000`
- `payload[0].track_metadata.additional_info.media_player`: "Melovian"
- `payload[0].track_metadata.additional_info.submission_client`: "Melovian"

## Bundled extension packaging

- Bundled extensions live in `internal/extensions/bundled/`: `lastfm`, `listenbrainz`, `lyrics`, `lyrics-whisper`, `metadata`, `rocksky`. Each ships a `melovian-extension.json` manifest.
- Optional bundle registry: `internal/extensions/bundled.go` `BundledOptional` marks ids that are not installed by default (rocksky is one).
- `InstallBundled(dataDir)` copies bundled extensions into the user extensions directory, preserving uninstalled markers and skipping not-yet-installed optional ones.
