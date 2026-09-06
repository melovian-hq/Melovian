// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { coverArtUrl } from "$lib/subsonic";
import {
  coverArtFallbackUrl,
  trackCoverPaletteKey,
  trackCoverSeed,
} from "./cover-art-fallback";
import type { SubsonicConfig } from "$lib/subsonic/types";
import type { SubsonicSong } from "$lib/subsonic/types";

export interface MediaSessionHandlers {
  onPlay: () => void | Promise<void>;
  onPause: () => void | Promise<void>;
  onPrevious: () => void | Promise<void>;
  onNext: () => void | Promise<void>;
  onSeek: (seconds: number) => void | Promise<void>;
}

const MEDIA_SESSION_ACTIONS = [
  "play",
  "pause",
  "previoustrack",
  "nexttrack",
  "seekto",
] as const;

function bindActionHandlers(handlers: MediaSessionHandlers) {
  try {
    navigator.mediaSession.setActionHandler("play", () => {
      void handlers.onPlay();
    });
    navigator.mediaSession.setActionHandler("pause", () => {
      void handlers.onPause();
    });
    navigator.mediaSession.setActionHandler("previoustrack", () => {
      void handlers.onPrevious();
    });
    navigator.mediaSession.setActionHandler("nexttrack", () => {
      void handlers.onNext();
    });
    navigator.mediaSession.setActionHandler("seekto", (details) => {
      if (details.seekTime != null) {
        void handlers.onSeek(details.seekTime);
      }
    });
  } catch {
    /* unsupported action */
  }
}

function clearActionHandlers() {
  for (const action of MEDIA_SESSION_ACTIONS) {
    try {
      navigator.mediaSession.setActionHandler(action, null);
    } catch {
      /* unsupported action */
    }
  }
}

export function bindMediaSession(
  track: SubsonicSong,
  config: SubsonicConfig,
  playing: boolean,
  handlers: MediaSessionHandlers,
) {
  if (typeof navigator === "undefined" || !("mediaSession" in navigator))
    return;

  const artId = track.coverArt ?? track.albumId ?? track.id;
  const artwork: MediaImage[] = [];
  for (const size of [96, 192, 512]) {
    const src = coverArtUrl(config, artId, size);
    if (src)
      artwork.push({ src, sizes: `${size}x${size}`, type: "image/jpeg" });
  }
  artwork.push({
    src: coverArtFallbackUrl(
      trackCoverSeed(track),
      trackCoverPaletteKey(track),
    ),
    sizes: "512x512",
    type: "image/svg+xml",
  });

  bindActionHandlers(handlers);

  navigator.mediaSession.metadata = new MediaMetadata({
    title: track.title,
    artist: track.artist ?? "Unknown artist",
    album: track.album ?? "",
    artwork,
  });

  navigator.mediaSession.playbackState = playing ? "playing" : "paused";
}

export function updateMediaSessionPosition(
  duration: number,
  position: number,
  playbackRate = 1,
) {
  if (typeof navigator === "undefined" || !("mediaSession" in navigator))
    return;
  if (!navigator.mediaSession.setPositionState) return;
  if (!Number.isFinite(duration) || duration <= 0) return;

  const safePosition = Math.max(0, Math.min(position, duration));
  if (!Number.isFinite(safePosition)) return;

  try {
    navigator.mediaSession.setPositionState({
      duration,
      playbackRate: Number.isFinite(playbackRate) ? playbackRate : 1,
      position: safePosition,
    });
  } catch {
    /* ignore invalid transitions on mobile browsers */
  }
}

export function clearMediaSession() {
  if (typeof navigator === "undefined" || !("mediaSession" in navigator))
    return;
  clearActionHandlers();
  navigator.mediaSession.metadata = null;
  navigator.mediaSession.playbackState = "none";
}
