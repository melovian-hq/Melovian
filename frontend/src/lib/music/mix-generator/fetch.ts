// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  clusterArtistFeatures,
  dedupeTracks,
  normalizeMixKey,
  type ArtistCluster,
  type ArtistFeature,
} from "../mix-select";
import { decadeFromYear } from "../taste-score";
import type {
  SubsonicAlbum,
  SubsonicGenre,
  SubsonicSong,
} from "$lib/subsonic/types";
import { hashString, topArtistLabels } from "./helpers";
import type { MixBuildContext, MixBuildState, MixFetchers } from "./types";

export async function loadArtistFeatures(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  labels: readonly string[],
): Promise<ArtistFeature[]> {
  const albumLists = await Promise.all(
    labels.map((label) =>
      fetchers.searchArtistAlbums(label, 6).catch(() => [] as SubsonicAlbum[]),
    ),
  );

  const weightByLabel = new Map<string, number>();
  for (const [index, artist] of (ctx.stats?.topArtists ?? []).entries()) {
    weightByLabel.set(
      artist.label,
      Math.max(artist.count, labels.length - index),
    );
  }
  for (const entry of ctx.history) {
    const label = entry.artistName?.trim();
    if (!label) continue;
    weightByLabel.set(
      label,
      (weightByLabel.get(label) ?? 0) + Math.max(1, entry.playCount),
    );
  }

  return labels
    .map((label, index) => {
      const albums = albumLists[index] ?? [];
      const genres: string[] = [];
      const decades: number[] = [];
      for (const album of albums) {
        if (album.genre?.trim()) genres.push(album.genre);
        if (typeof album.year === "number" && album.year >= 1900) {
          decades.push(decadeFromYear(album.year));
        }
      }
      return {
        label,
        key: normalizeMixKey(label),
        weight: weightByLabel.get(label) ?? labels.length - index,
        genres,
        decades,
      };
    })
    .filter((feature) => feature.key.length > 0);
}

export async function getArtistClusters(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
): Promise<ArtistCluster[]> {
  if (state.artistClusters) return state.artistClusters;
  const labels = topArtistLabels(ctx, 15);
  if (labels.length === 0) {
    state.artistClusters = [];
    return [];
  }
  const features = await loadArtistFeatures(ctx, fetchers, labels);
  state.artistClusters = clusterArtistFeatures(
    features,
    `${ctx.daySeed}:clusters`,
    3,
    5,
  );
  return state.artistClusters;
}

export async function similarSongsForSeeds(
  fetchers: MixFetchers,
  seedIds: readonly string[],
  count: number,
): Promise<SubsonicSong[]> {
  if (seedIds.length === 0) return [];
  const groups = await Promise.all(
    seedIds.map((id) =>
      fetchers.getSimilarSongs(id, count).catch(() => [] as SubsonicSong[]),
    ),
  );
  return dedupeTracks(groups.flat());
}
export async function inferPersonalGenres(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
): Promise<Map<string, number>> {
  const scores = new Map<string, number>();
  const artists = ctx.stats?.topArtists.slice(0, 5) ?? [];
  if (artists.length === 0) return scores;

  const albumLists = await Promise.all(
    artists.map((artist, index) =>
      fetchers.searchArtistAlbums(artist.label, 4).then((albums) => ({
        albums,
        weight: artists.length - index,
      })),
    ),
  );

  for (const { albums, weight } of albumLists) {
    for (const album of albums) {
      const genre = album.genre?.trim();
      if (!genre) continue;
      scores.set(genre, (scores.get(genre) ?? 0) + weight);
    }
  }

  return scores;
}

/** Ranked genres for multi-slot genre mixes. */
export async function rankGenres(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  genres: SubsonicGenre[],
): Promise<SubsonicGenre[]> {
  const libraryRanked = [...genres]
    .filter((genre) => (genre.songCount ?? 0) > 0)
    .sort(
      (a, b) =>
        (b.songCount ?? 0) - (a.songCount ?? 0) || a.name.localeCompare(b.name),
    )
    .slice(0, 12);

  if (libraryRanked.length === 0) return [];

  const mode = ctx.settings.genreSelection;
  if (mode === "library") return libraryRanked;

  const personalScores = await inferPersonalGenres(ctx, fetchers);
  const personalRanked = [...personalScores.entries()]
    .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
    .map(([name]) => genres.find((genre) => genre.name === name))
    .filter(
      (genre): genre is SubsonicGenre => !!genre && (genre.songCount ?? 0) > 0,
    );

  if (mode === "personal") {
    return personalRanked.length > 0 ? personalRanked : libraryRanked;
  }

  const blended = new Map<string, SubsonicGenre>();
  for (const genre of personalRanked) {
    blended.set(genre.name, genre);
  }
  for (const genre of libraryRanked) {
    if (!blended.has(genre.name)) blended.set(genre.name, genre);
  }
  return [...blended.values()];
}

export async function pickGenre(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  genres: SubsonicGenre[],
): Promise<SubsonicGenre | null> {
  const ranked = await rankGenres(ctx, fetchers, genres);
  if (ranked.length === 0) return null;
  const index = hashString(`${ctx.daySeed}:genre`) % Math.min(ranked.length, 8);
  return ranked[index] ?? ranked[0];
}
export function createCachedFetchers(fetchers: MixFetchers): MixFetchers {
  const albumCache = new Map<string, SubsonicSong[]>();
  const artistSongCache = new Map<string, SubsonicSong[]>();
  const artistAlbumCache = new Map<string, SubsonicAlbum[]>();
  const similarCache = new Map<string, SubsonicSong[]>();
  const randomCache = new Map<number, SubsonicSong[]>();
  let genresCache: SubsonicGenre[] | undefined;
  let starredCache: SubsonicSong[] | undefined;

  return {
    ...fetchers,
    getAlbumSongs: async (albumId) => {
      const cached = albumCache.get(albumId);
      if (cached) return cached;
      const songs = await fetchers.getAlbumSongs(albumId);
      albumCache.set(albumId, songs);
      return songs;
    },
    searchArtistSongs: async (artist, limit) => {
      const key = artist.trim().toLowerCase();
      const cached = artistSongCache.get(key);
      if (cached && cached.length >= limit) return cached.slice(0, limit);
      const songs = await fetchers.searchArtistSongs(
        artist,
        Math.max(limit, cached?.length ?? 0),
      );
      artistSongCache.set(key, songs);
      return songs.slice(0, limit);
    },
    searchArtistAlbums: async (artist, limit) => {
      const key = artist.trim().toLowerCase();
      const cached = artistAlbumCache.get(key);
      if (cached && cached.length >= limit) return cached.slice(0, limit);
      const albums = await fetchers.searchArtistAlbums(
        artist,
        Math.max(limit, cached?.length ?? 0),
      );
      artistAlbumCache.set(key, albums);
      return albums.slice(0, limit);
    },
    getSimilarSongs: async (trackId, count) => {
      const key = `${trackId}:${count}`;
      const cached = similarCache.get(key);
      if (cached) return cached;
      const songs = await fetchers.getSimilarSongs(trackId, count);
      similarCache.set(key, songs);
      return songs;
    },
    getRandomSongs: async (count) => {
      const cached = randomCache.get(count);
      if (cached) return cached;
      const songs = await fetchers.getRandomSongs(count);
      randomCache.set(count, songs);
      return songs;
    },
    getGenres: async () => {
      if (genresCache) return genresCache;
      genresCache = await fetchers.getGenres();
      return genresCache;
    },
    getStarredSongs: async () => {
      if (starredCache) return starredCache;
      starredCache = await fetchers.getStarredSongs();
      return starredCache;
    },
  };
}

export async function hydrateStarred(
  state: MixBuildState,
  fetchers: MixFetchers,
): Promise<void> {
  const starred = await fetchers
    .getStarredSongs()
    .catch(() => [] as SubsonicSong[]);
  for (const track of starred) {
    if (track.id) state.starredTrackIds.add(track.id);
  }
}
