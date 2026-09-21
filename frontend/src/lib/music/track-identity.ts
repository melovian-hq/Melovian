// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Track identity is normalized title + artist, so the same song on a
 * remaster, single, or compilation counts as one pick in mixes and radio.
 */
export function trackIdentityKey(track: {
  id?: string;
  title?: string;
  artist?: string;
}): string {
  const title = track.title?.trim().toLowerCase() ?? "";
  if (!title) return `id:${track.id ?? ""}`;
  const artist = track.artist?.trim().toLowerCase() ?? "";
  return `${title}|${artist}`;
}

/** Drop identity duplicates, keeping the first occurrence. */
export function dedupeTrackIdentities<
  T extends { id?: string; title?: string; artist?: string },
>(tracks: readonly T[]): T[] {
  const seen = new Set<string>();
  const result: T[] = [];
  for (const track of tracks) {
    const key = trackIdentityKey(track);
    if (seen.has(key)) continue;
    seen.add(key);
    result.push(track);
  }
  return result;
}
