// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { isLocalTrackId } from "$lib/music/source.svelte";

export function isRemoteCacheableTrackId(trackId: string): boolean {
  if (!trackId || isLocalTrackId(trackId)) return false;
  if (trackId.startsWith("open:")) return false;
  return true;
}

export function isRemoteCacheableTrack(track: {
  id: string;
  isInternetRadio?: boolean;
}): boolean {
  if (track.isInternetRadio) return false;
  return isRemoteCacheableTrackId(track.id);
}

export function selectTracksForOfflineDownload<
  T extends { id: string; isInternetRadio?: boolean },
>(tracks: readonly T[], downloadedIds: ReadonlySet<string>): T[] {
  const seen = new Set<string>();
  const selected: T[] = [];
  for (const track of tracks) {
    if (!isRemoteCacheableTrack(track)) continue;
    if (downloadedIds.has(track.id)) continue;
    if (seen.has(track.id)) continue;
    seen.add(track.id);
    selected.push(track);
  }
  return selected;
}
