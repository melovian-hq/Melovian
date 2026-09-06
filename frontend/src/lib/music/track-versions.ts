// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicSong } from "$lib/subsonic/types";

function normalizeTrackTitle(title: string): string {
  return title
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

export function trackQualityScore(
  track: Pick<
    SubsonicSong,
    "suffix" | "bitRate" | "contentType" | "transcoded"
  >,
): number {
  if (track.transcoded) return 50 + (track.bitRate ?? 0);

  const suffix = track.suffix?.toLowerCase() ?? "";
  const contentType = track.contentType?.toLowerCase() ?? "";
  const bitRate = track.bitRate ?? 0;

  if (suffix === "flac" || contentType.includes("flac")) return 1_000 + bitRate;
  if (suffix === "alac" || contentType.includes("alac")) return 950 + bitRate;
  if (suffix === "wav" || contentType.includes("wav")) return 900 + bitRate;
  if (suffix === "opus" || contentType.includes("opus")) return 600 + bitRate;
  if (suffix === "ogg" || contentType.includes("ogg")) return 550 + bitRate;
  if (suffix === "aac" || suffix === "m4a" || contentType.includes("aac")) {
    return 400 + bitRate;
  }
  if (suffix === "mp3" || contentType.includes("mpeg")) return 200 + bitRate;
  return 100 + bitRate;
}

export function trackVersionKey(
  track: SubsonicSong,
  fallbackIndex: number,
): string {
  const trackNumber = track.track ?? fallbackIndex + 1;
  return `${trackNumber}:${normalizeTrackTitle(track.title)}`;
}

export interface TrackVersionGroup {
  key: string;
  versions: SubsonicSong[];
  primary: SubsonicSong;
  displayIndex: number;
}

export function groupTrackVersions(songs: SubsonicSong[]): TrackVersionGroup[] {
  const map = new Map<string, SubsonicSong[]>();
  const order: string[] = [];
  songs.forEach((song, index) => {
    const key = trackVersionKey(song, index);
    if (!map.has(key)) order.push(key);
    const list = map.get(key) ?? [];
    list.push(song);
    map.set(key, list);
  });

  return order.map((key, groupIndex) => {
    const versions = [...(map.get(key) ?? [])].sort(
      (a, b) => trackQualityScore(b) - trackQualityScore(a),
    );
    const primary = versions[0]!;
    return {
      key,
      versions,
      primary,
      displayIndex: primary.track ?? groupIndex + 1,
    };
  });
}

export function albumPlaybackTracks(songs: SubsonicSong[]): SubsonicSong[] {
  return groupTrackVersions(songs).map((group) => group.primary);
}

export function indexInAlbumSongs(
  songs: SubsonicSong[],
  trackId: string,
): number {
  return songs.findIndex((song) => song.id === trackId);
}
