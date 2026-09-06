// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

const UNKNOWN_ARTIST_KEYS = new Set([
  "",
  "unknown",
  "unknown artist",
  "unknownartist",
]);

const UNKNOWN_ALBUM_KEYS = new Set([
  "",
  "unknown",
  "unknown album",
  "unknownalbum",
]);

export function unknownMetadataKey(value: string | undefined | null): string {
  return (value ?? "")
    .trim()
    .toLowerCase()
    .replace(/^\[+|\]+$/g, "")
    .replace(/^<+|>+$/g, "")
    .replace(/\s+/g, " ")
    .trim();
}

export function isUnknownArtistName(name: string | undefined | null): boolean {
  return UNKNOWN_ARTIST_KEYS.has(unknownMetadataKey(name));
}

export function isUnknownAlbumName(name: string | undefined | null): boolean {
  return UNKNOWN_ALBUM_KEYS.has(unknownMetadataKey(name));
}

export function albumHasUnknownMetadata(album: {
  name?: string;
  artist?: string;
}): boolean {
  return isUnknownArtistName(album.artist) || isUnknownAlbumName(album.name);
}

export function artistHasUnknownMetadata(artist: { name?: string }): boolean {
  return isUnknownArtistName(artist.name);
}

export function trackHasUnknownMetadata(track: {
  artist?: string;
  album?: string;
}): boolean {
  return isUnknownArtistName(track.artist) || isUnknownAlbumName(track.album);
}

export function rejectUnknownAlbums<
  T extends { name?: string; artist?: string },
>(items: readonly T[], hide: boolean): T[] {
  if (!hide) return items as T[];
  return items.filter((item) => !albumHasUnknownMetadata(item));
}

export function rejectUnknownArtists<T extends { name?: string }>(
  items: readonly T[],
  hide: boolean,
): T[] {
  if (!hide) return items as T[];
  return items.filter((item) => !artistHasUnknownMetadata(item));
}

export function rejectUnknownTracks<
  T extends { artist?: string; album?: string },
>(items: readonly T[], hide: boolean): T[] {
  if (!hide) return items as T[];
  return items.filter((item) => !trackHasUnknownMetadata(item));
}

export function rejectUnknownListenEvents<
  T extends { artistName?: string; albumTitle?: string },
>(items: readonly T[], hide: boolean): T[] {
  if (!hide) return items as T[];
  return items.filter(
    (item) =>
      !trackHasUnknownMetadata({
        artist: item.artistName,
        album: item.albumTitle,
      }),
  );
}
