// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicSong, SubsonicTrackArtist } from "$lib/subsonic";
import { artistIdFromName } from "$lib/music/local-ids";

export interface TrackArtistCredit {
  id?: string;
  name: string;
}

export interface TrackAlbumCredit {
  id?: string;
  name: string;
}

const ARTIST_CREDIT_SPLIT =
  /\s*(?:,|;|\s&\s|\s+feat\.?\s+|\s+ft\.?\s+|\s+featuring\s+|\s+x\s+|\s+vs\.?\s+)\s*/i;

export function splitArtistNames(artist: string): string[] {
  const trimmed = artist.trim();
  if (!trimmed) return [];
  const parts = trimmed
    .split(ARTIST_CREDIT_SPLIT)
    .map((part) => part.trim())
    .filter(Boolean);
  return parts.length > 0 ? parts : [trimmed];
}

export function trackHasFullMetadata(track: SubsonicSong): boolean {
  return Boolean(
    track.title?.trim() && track.artist?.trim() && (track.duration ?? 0) > 0,
  );
}

export function trackNeedsLinkMetadata(track: SubsonicSong): boolean {
  const artist = track.artist?.trim();
  const album = track.album?.trim();
  if (artist && !track.artistId && !hasStructuredArtists(track)) return true;
  if (album && !track.albumId) return true;
  if (
    artist &&
    splitArtistNames(artist).length > 1 &&
    !hasStructuredArtists(track)
  ) {
    return true;
  }
  return false;
}

function hasStructuredArtists(track: SubsonicSong): boolean {
  return (track.artists?.length ?? 0) > 0;
}

function creditsFromStructuredArtists(
  artists: SubsonicTrackArtist[],
): TrackArtistCredit[] {
  return artists
    .map((artist) => ({
      id: artist.id || undefined,
      name: artist.name.trim(),
    }))
    .filter((artist) => artist.name.length > 0);
}

export function resolveTrackArtists(
  track: SubsonicSong,
  options: { localIds?: boolean } = {},
): TrackArtistCredit[] {
  if (track.artists?.length) {
    const credits = creditsFromStructuredArtists(track.artists);
    if (credits.length > 0) return credits;
  }

  const artist = track.artist?.trim();
  if (!artist) return [{ name: "Unknown" }];

  const names = splitArtistNames(artist);
  if (names.length === 1) {
    return [
      {
        id:
          track.artistId ||
          (options.localIds ? artistIdFromName(names[0]) : undefined),
        name: names[0],
      },
    ];
  }

  return names.map((name, index) => ({
    id: options.localIds
      ? artistIdFromName(name)
      : index === 0
        ? track.artistId
        : undefined,
    name,
  }));
}

export function resolveTrackAlbum(
  track: SubsonicSong,
): TrackAlbumCredit | null {
  const name = track.album?.trim();
  if (!name) return null;
  return {
    id: track.albumId,
    name,
  };
}
