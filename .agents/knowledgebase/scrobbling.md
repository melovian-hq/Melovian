# Scrobbling

Melovian can submit listens to Rocksky through the bundled Rocksky extension.

## Rocksky client

`internal/rocksky/client.go` posts to `https://audioscrobbler.rocksky.app/1/submit-listens`.

It provides three calls:

- `SubmitNowPlaying(ctx, token, track)` sends `listen_type: "playing_now"`.
- `SubmitScrobble(ctx, token, track, listenedAt)` sends `listen_type: "single"` with a Unix timestamp.
- `ValidateToken(ctx, token)` checks a token against `/1/validate-token`.

Requests use `Authorization: Token <token>` and a JSON body.

## Submission payload fields

- `listen_type`: `playing_now` or `single`
- `payload[0].listened_at`: Unix timestamp for `single`
- `payload[0].track_metadata.artist_name`
- `payload[0].track_metadata.track_name`
- `payload[0].track_metadata.release_name`
- `payload[0].track_metadata.additional_info.duration_ms`: `track.DurationSeconds * 1000`
- `payload[0].track_metadata.additional_info.media_player`: "Melovian"
- `payload[0].track_metadata.additional_info.submission_client`: "Melovian"

## Bundled extension packaging

- Manifest: `internal/extensions/bundled/rocksky/melovian-extension.json`
- Optional bundle registry: `internal/extensions/bundled.go` sets `BundledOptional["rocksky"] = true`.
- `InstallBundled(dataDir)` copies bundled extensions into the user extensions directory, preserving uninstalled markers and skipping not-yet-installed optional ones.
