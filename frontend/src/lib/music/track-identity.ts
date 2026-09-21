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

const LOSSLESS_FORMAT =
  /(?:^|[/.\s-])(flac|wav|aiff?|alac|ape|wv|dsf|dff|pcm|caf)$/i;

/**
 * Rough fidelity score for one track variant. Lossless containers dominate,
 * then bitrate, then bit depth and sample rate.
 */
export function trackQualityScore(track: {
  bitRate?: number;
  samplingRate?: number;
  bitDepth?: number;
  suffix?: string;
  contentType?: string;
}): number {
  const format = `${track.suffix ?? ""} ${track.contentType ?? ""}`;
  const lossless = LOSSLESS_FORMAT.test(format) ? 1_000_000 : 0;
  const bitRate = Math.max(0, track.bitRate ?? 0);
  const bitDepth = Math.min(32, Math.max(0, track.bitDepth ?? 0));
  const sampleRate = Math.min(192_000, Math.max(0, track.samplingRate ?? 0));
  return lossless + bitRate * 100 + bitDepth * 10 + sampleRate / 1000;
}

/**
 * Collapse same-song duplicates keeping the preferred encode. With
 * preferHighQuality false (low bandwidth mode) the smaller lossy file wins.
 * Surviving picks keep their input order.
 */
export function dedupeTracksByQuality<
  T extends {
    id?: string;
    title?: string;
    artist?: string;
    bitRate?: number;
    samplingRate?: number;
    bitDepth?: number;
    suffix?: string;
    contentType?: string;
  },
>(tracks: readonly T[], preferHighQuality = true): T[] {
  const best = new Map<string, { track: T; score: number; index: number }>();
  tracks.forEach((track, index) => {
    const key = trackIdentityKey(track);
    const score = trackQualityScore(track) * (preferHighQuality ? 1 : -1);
    const existing = best.get(key);
    if (!existing || score > existing.score) {
      best.set(key, { track, score, index });
    }
  });
  return [...best.values()]
    .sort((a, b) => a.index - b.index)
    .map((entry) => entry.track);
}
