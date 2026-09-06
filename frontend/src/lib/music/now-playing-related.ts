// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import {
  getRelatedTracksCached,
  peekRelatedTracks,
} from "$lib/music/related-tracks-cache";
import type { SubsonicSong } from "$lib/subsonic";

export type RelatedTracksSyncArgs = {
  track: SubsonicSong | null | undefined;
  activeTab: string;
  library: MusicLibraryAdapter;
  limit?: number;
  onTracks: (tracks: SubsonicSong[]) => void;
  onLoading: (loading: boolean) => void;
};

/**
 * Body for the now-playing related-tracks $effect.
 * Returns a teardown that cancels in-flight updates, or undefined.
 */
export function syncRelatedTracks(
  args: RelatedTracksSyncArgs,
): (() => void) | undefined {
  const {
    track: current,
    activeTab,
    library,
    limit = 24,
    onTracks,
    onLoading,
  } = args;

  if (!current) {
    onTracks([]);
    onLoading(false);
    return;
  }

  const cached =
    activeTab === "related" ? peekRelatedTracks(current, limit) : null;
  if (cached) {
    onTracks(cached);
    onLoading(false);
  }

  if (activeTab !== "related") {
    onLoading(false);
    return;
  }

  if (!cached) onLoading(true);

  let cancelled = false;
  void getRelatedTracksCached(library, current, limit)
    .then((songs) => {
      if (cancelled) return;
      onTracks(songs);
    })
    .catch(() => {
      if (!cancelled && !cached) onTracks([]);
    })
    .finally(() => {
      if (!cancelled) onLoading(false);
    });

  return () => {
    cancelled = true;
  };
}
