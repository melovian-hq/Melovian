// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import type { SubsonicSong } from "$lib/subsonic";

function dedupeTracks(
  tracks: SubsonicSong[],
  excludeId: string,
  limit: number,
): SubsonicSong[] {
  const seen = new Set<string>();
  const out: SubsonicSong[] = [];
  for (const track of tracks) {
    if (!track.id || track.id === excludeId || seen.has(track.id)) continue;
    seen.add(track.id);
    out.push(track);
    if (out.length >= limit) break;
  }
  return out;
}

function artistMatches(track: SubsonicSong, artistName: string): boolean {
  const needle = artistName.toLowerCase();
  if (track.artist?.toLowerCase() === needle) return true;
  return (
    track.artists?.some((artist) => artist.name.toLowerCase() === needle) ??
    false
  );
}

function collectUntilLimit(
  collected: SubsonicSong[],
  tracks: SubsonicSong[],
  excludeId: string,
  limit: number,
): boolean {
  for (const track of tracks) {
    if (!track.id || track.id === excludeId) continue;
    if (collected.some((item) => item.id === track.id)) continue;
    collected.push(track);
    if (collected.length >= limit) return true;
  }
  return collected.length >= limit;
}

async function loadFallbackRelatedTracks(
  library: MusicLibraryAdapter,
  track: SubsonicSong,
  limit: number,
): Promise<SubsonicSong[]> {
  const collected: SubsonicSong[] = [];

  const albumPromise = track.albumId
    ? library.getAlbum(track.albumId).catch(() => null)
    : Promise.resolve(null);

  const artistPromise = track.artistId
    ? library.getArtist(track.artistId).catch(() => null)
    : Promise.resolve(null);

  const [album, artistDetail] = await Promise.all([
    albumPromise,
    artistPromise,
  ]);

  if (album) {
    if (collectUntilLimit(collected, album.songs, track.id, limit)) {
      return dedupeTracks(collected, track.id, limit);
    }
  }

  if (artistDetail) {
    const albumIds = artistDetail.albums.slice(0, 4).map((album) => album.id);
    const albumDetails = await Promise.all(
      albumIds.map((id) => library.getAlbum(id).catch(() => null)),
    );
    for (const detail of albumDetails) {
      if (!detail) continue;
      if (collectUntilLimit(collected, detail.songs, track.id, limit)) {
        return dedupeTracks(collected, track.id, limit);
      }
    }
  }

  const artistName = track.artist?.trim();
  if (artistName && library.search3) {
    const search = await Promise.resolve(
      library.search3(artistName, limit),
    ).catch(() => null);
    if (search) {
      collectUntilLimit(
        collected,
        search.songs.filter((song) => artistMatches(song, artistName)),
        track.id,
        limit,
      );
    }
  }

  return dedupeTracks(collected, track.id, limit);
}

export async function loadRelatedTracks(
  library: MusicLibraryAdapter,
  track: SubsonicSong,
  limit = 24,
): Promise<SubsonicSong[]> {
  const similar = await library
    .getSimilarSongs(track.id, limit)
    .catch(() => []);
  const fromSimilar = dedupeTracks(similar, track.id, limit);
  if (fromSimilar.length > 0) {
    return fromSimilar;
  }

  return loadFallbackRelatedTracks(library, track, limit);
}
