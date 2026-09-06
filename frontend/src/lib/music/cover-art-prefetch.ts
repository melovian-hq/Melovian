// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { coverArtUrl } from "$lib/subsonic";
import type { SubsonicConfig, SubsonicSong } from "$lib/subsonic";
import { createBoundedSet } from "$lib/core/bounded-cache";
import { COVER_SIZE_NOW_PLAYING, COVER_SIZE_PLAYER } from "./cover-art-sizes";

const LOADED_MAX_ENTRIES = 150;

const loaded = createBoundedSet<string>(LOADED_MAX_ENTRIES);
const inflight = new Map<string, Promise<void>>();

export function trackCoverArtId(track: SubsonicSong): string | undefined {
  return track.coverArt ?? track.albumId ?? track.id;
}

export function trackCoverArtUrl(
  config: SubsonicConfig,
  track: SubsonicSong,
  size: number,
): string | null {
  const id = trackCoverArtId(track);
  if (!id) return null;
  return coverArtUrl(config, id, size, track.id);
}

export function isCoverArtPrefetched(url: string | null | undefined): boolean {
  return Boolean(url && loaded.has(url));
}

export function prefetchCoverArt(url: string | null | undefined): void {
  if (!url || loaded.has(url) || inflight.has(url)) return;

  const promise = new Promise<void>((resolve) => {
    const img = new Image();
    img.decoding = "async";
    img.onload = () => {
      loaded.add(url);
      resolve();
    };
    img.onerror = () => resolve();
    img.src = url;
  });

  inflight.set(url, promise);
  void promise.finally(() => {
    inflight.delete(url);
  });
}

export function prefetchTrackCoverArt(
  config: SubsonicConfig,
  track: SubsonicSong,
  size = COVER_SIZE_NOW_PLAYING,
): void {
  prefetchCoverArt(trackCoverArtUrl(config, track, size));
}

export function prefetchNowPlayingCoverArt(
  config: SubsonicConfig,
  track: SubsonicSong,
): void {
  prefetchCoverArt(trackCoverArtUrl(config, track, COVER_SIZE_PLAYER));
  prefetchCoverArt(trackCoverArtUrl(config, track, COVER_SIZE_NOW_PLAYING));
}
