// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  dedupeTracks,
  interleaveByArtist,
  mixFitScore,
  normalizeMixKey,
  seedFromCluster,
  tracksMatchSeed,
  defaultMixPolicy,
  type MixSeedProfile,
} from "../mix-select";
import {
  decadeFromYear,
  isTopArtistTrack,
  rankedDecadeKeys,
  topAffinityKeys,
} from "../taste-score";
import type { SubsonicAlbum, SubsonicSong } from "$lib/subsonic/types";
import { getArtistClusters, similarSongsForSeeds, rankGenres } from "./fetch";
import {
  albumYearIndex,
  clusterSeedIds,
  dedupeAlbums,
  finalizeMix,
  gradientFromGenre,
  gradientFromSeed,
  historyByListenWeight,
  isDiscoverCandidate,
  pickAlbumCover,
  pickDeepCutTracks,
  rankedDecades,
  seedFromArtists,
  topArtistLabels,
  trackMatchesDecade,
  weightedSimilarSeeds,
} from "./helpers";
import { selectContext } from "./context";
import type {
  GeneratedMix,
  MixBuildContext,
  MixBuildState,
  MixFetchers,
} from "./types";

function buildOnRepeatMix(
  ctx: MixBuildContext,
  state: MixBuildState,
): GeneratedMix | null {
  const recentCutoff = Date.now() - 14 * 86_400_000;
  const ranked = historyByListenWeight(ctx.history).filter(
    (entry) =>
      entry.playCount >= 2 &&
      new Date(entry.lastPlayedAt).getTime() >= recentCutoff &&
      !ctx.skippedTrackIds.has(entry.trackId),
  );
  const minTracks = ctx.settings.minTracksPerMix;
  if (ranked.length < minTracks) return null;

  const tracks = ranked.slice(0, minTracks + 24).map(ctx.entryToSong);
  const topArtist = ranked[0]?.artistName;
  return finalizeMix(
    {
      id: "on-repeat",
      title: "On Repeat",
      subtitle: topArtist
        ? `${topArtist} and your most-played tracks`
        : "Tracks you cannot stop playing",
      gradient: gradientFromSeed(`${ctx.daySeed}:on-repeat`),
    },
    tracks,
    `${ctx.daySeed}:on-repeat:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(
        ranked.slice(0, 8).map((entry) => entry.artistName),
      ),
      policy: { familiarRatio: 0.8, discoverRatio: 0.1, penalizeRecent: false },
    },
  );
}

function buildReplayMix(
  ctx: MixBuildContext,
  state: MixBuildState,
): GeneratedMix | null {
  const recent = ctx.history
    .slice(0, 60)
    .filter(
      (entry) =>
        entry.playCount < 10 && !ctx.skippedTrackIds.has(entry.trackId),
    )
    .map(ctx.entryToSong);
  const minTracks = ctx.settings.minTracksPerMix;
  if (recent.length < minTracks) return null;

  return finalizeMix(
    {
      id: "replay",
      title: "Replay Mix",
      subtitle: "Recent favorites on rotation",
      gradient: gradientFromSeed(`${ctx.daySeed}:replay`),
    },
    dedupeTracks(recent),
    `${ctx.daySeed}:replay:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(
        recent.slice(0, 10).map((track) => track.artist ?? ""),
      ),
      policy: { familiarRatio: 0.7, discoverRatio: 0.2 },
    },
  );
}

async function buildDailyArtistMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
  slot: number,
): Promise<GeneratedMix | null> {
  const clusters = await getArtistClusters(ctx, fetchers, state);
  const cluster = clusters[slot];
  if (!cluster || cluster.artists.length === 0) return null;

  const seed = seedFromCluster(cluster);
  const perArtist =
    Math.ceil(
      ctx.settings.minTracksPerMix / Math.max(1, cluster.artists.length),
    ) + 4;

  const [artistSongGroups, similar, starred] = await Promise.all([
    Promise.all(
      cluster.artists.map((artist) =>
        fetchers.searchArtistSongs(artist.label, perArtist),
      ),
    ),
    similarSongsForSeeds(
      fetchers,
      clusterSeedIds(ctx, cluster, `${ctx.daySeed}:daily:${slot}:seeds`, 4),
      12,
    ),
    fetchers.getStarredSongs().catch(() => [] as SubsonicSong[]),
  ]);

  const clusterSongs: SubsonicSong[] = [];
  for (const [i, artist] of cluster.artists.entries()) {
    const matches = artistSongGroups[i].filter(
      (song) => normalizeMixKey(song.artist) === artist.key,
    );
    clusterSongs.push(...matches);
  }

  const historyTracks = historyByListenWeight(ctx.history)
    .filter((entry) => seed.artistKeys.has(normalizeMixKey(entry.artistName)))
    .slice(0, 16)
    .map(ctx.entryToSong);

  const related = similar.filter(
    (track) => tracksMatchSeed(track, seed) || !track.genre || !track.artist,
  );
  const starredInCluster = starred.filter((track) =>
    tracksMatchSeed(track, seed),
  );

  const pool = dedupeTracks([
    ...clusterSongs,
    ...historyTracks,
    ...starredInCluster,
    ...related,
  ]);

  const names = cluster.artists.map((artist) => artist.label).join(", ");
  const genreLabel = cluster.genreKeys[0];
  const subtitle = genreLabel ? `${genreLabel} · ${names}` : names;

  return finalizeMix(
    {
      id: `daily-mix-${slot + 1}`,
      title: `Daily Mix ${slot + 1}`,
      subtitle,
      gradient: gradientFromSeed(`${ctx.daySeed}:daily:${slot}`),
    },
    pool,
    `${ctx.daySeed}:daily:${slot}:flow`,
    ctx,
    state,
    {
      seed,
      policy: {
        familiarRatio: 0.5,
        discoverRatio: 0.32,
        maxPerArtist: 4,
        maxPerAlbum: 2,
      },
    },
  );
}

async function buildGenreMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
  slot: number,
): Promise<GeneratedMix | null> {
  const genres = await fetchers.getGenres();
  const ranked = await rankGenres(ctx, fetchers, genres);
  const genre = ranked[slot];
  if (!genre) return null;

  const genreKey = normalizeMixKey(genre.name);
  const seed: MixSeedProfile = {
    artistKeys: new Set(topArtistLabels(ctx, 8).map(normalizeMixKey)),
    genreKeys: new Set(genreKey ? [genreKey] : []),
    decades: new Set(),
    artistFocused: false,
  };

  const genreSongs = await fetchers.getGenreSongs(
    genre.name,
    Math.max(48, ctx.settings.minTracksPerMix + 16),
  );
  const similar = await similarSongsForSeeds(
    fetchers,
    weightedSimilarSeeds(
      ctx.stats,
      ctx.history,
      `${ctx.daySeed}:genre-seeds:${slot}`,
      3,
    ),
    10,
  );
  const related = similar.filter(
    (song) =>
      isDiscoverCandidate(song, ctx) &&
      (tracksMatchSeed(song, seed) || !song.genre),
  );

  return finalizeMix(
    {
      id: `genre-mix-${slot + 1}`,
      title: `${genre.name} Mix`,
      subtitle: `A ${genre.name.toLowerCase()} session for today`,
      gradient: gradientFromGenre(genre.name),
    },
    dedupeTracks([...genreSongs, ...related]),
    `${ctx.daySeed}:genre:${slot}:flow`,
    ctx,
    state,
    {
      seed,
      policy: {
        familiarRatio: 0.45,
        discoverRatio: 0.4,
        exploreBonus: 0.5,
        maxPerArtist: 3,
      },
    },
  );
}

async function buildDiscoverMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
): Promise<GeneratedMix | null> {
  const seedIds = weightedSimilarSeeds(
    ctx.stats,
    ctx.history,
    `${ctx.daySeed}:discover-seeds`,
    8,
  );

  const midTier = historyByListenWeight(ctx.history)
    .filter((entry) => {
      const plays = entry.playCount;
      return (
        plays >= 2 &&
        plays <= 14 &&
        !ctx.recentlyPlayedIds.has(entry.trackId) &&
        !ctx.skippedTrackIds.has(entry.trackId)
      );
    })
    .slice(0, 12)
    .map((entry) => entry.trackId);
  const explorationSeeds = [
    ...new Set([...seedIds.slice(0, 5), ...midTier.slice(0, 6)]),
  ].slice(0, 12);

  const [similar, randomSongs, randomAlbums, newestAlbums] = await Promise.all([
    similarSongsForSeeds(fetchers, explorationSeeds, 24),
    fetchers.getRandomSongs(48),
    fetchers.getRandomAlbums(6),
    fetchers.getNewestAlbums(10).catch(() => [] as SubsonicAlbum[]),
  ]);

  const newestSongs = (
    await Promise.all(
      newestAlbums.slice(0, 5).map((album) => fetchers.getAlbumSongs(album.id)),
    )
  ).flat();

  const adjacent = similar.filter((track) => isDiscoverCandidate(track, ctx));
  const fresh = dedupeTracks([...randomSongs, ...newestSongs]).filter((track) =>
    isDiscoverCandidate(track, ctx),
  );
  const nostalgic = historyByListenWeight(ctx.history)
    .filter(
      (entry) =>
        entry.playCount >= 3 &&
        entry.playCount <= 20 &&
        !ctx.recentlyPlayedIds.has(entry.trackId) &&
        !ctx.skippedTrackIds.has(entry.trackId),
    )
    .slice(0, 8)
    .map(ctx.entryToSong);

  const seed = seedFromArtists(
    topArtistLabels(ctx, 10),
    topAffinityKeys(ctx.tasteProfile.genreAffinity, 8),
    rankedDecadeKeys(ctx.tasteProfile.decadeAffinity, 4),
  );

  const scoredFresh = fresh
    .map((track) => ({
      track,
      score: mixFitScore(track, selectContext(ctx, state), seed, {
        ...defaultMixPolicy(14, false),
        exploreBonus: 0.95,
        penalizeRecent: true,
      }),
    }))
    .sort((a, b) => b.score - a.score)
    .slice(0, 20)
    .map((item) => item.track);

  const pool = dedupeTracks([...adjacent, ...scoredFresh, ...nostalgic]);
  if (pool.length < ctx.settings.minTracksPerMix) return null;

  return finalizeMix(
    {
      id: "discover",
      title: "Discover Weekly",
      subtitle: "Fresh picks and rediscoveries from your library",
      coverArtId: randomAlbums[0]?.coverArt ?? newestAlbums[0]?.coverArt,
      gradient: gradientFromSeed(`${ctx.daySeed}:discover`),
    },
    pool,
    `${ctx.daySeed}:discover:flow`,
    ctx,
    state,
    {
      seed,
      policy: {
        familiarRatio: 0.18,
        discoverRatio: 0.58,
        exploreBonus: 0.8,
        maxPerArtist: 2,
        maxPerAlbum: 2,
        penalizeRecent: true,
      },
    },
  );
}

async function buildDeepCutsMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
): Promise<GeneratedMix | null> {
  const labels = topArtistLabels(ctx, 12).slice(2, 8);
  if (labels.length === 0) return null;

  const albumLists = await Promise.all(
    labels.map((label) => fetchers.searchArtistAlbums(label, 5)),
  );

  const seenAlbumIds = new Set<string>();
  const albumIds: string[] = [];
  for (const albums of albumLists) {
    for (const album of albums) {
      if (seenAlbumIds.has(album.id)) continue;
      seenAlbumIds.add(album.id);
      albumIds.push(album.id);
      if (albumIds.length >= 8) break;
    }
    if (albumIds.length >= 8) break;
  }

  const albumSongLists = await Promise.all(
    albumIds.map((albumId) => fetchers.getAlbumSongs(albumId)),
  );

  const pool: SubsonicSong[] = [];
  for (const songs of albumSongLists) {
    pool.push(
      ...pickDeepCutTracks(
        songs,
        ctx,
        `${ctx.daySeed}:deep:${songs[0]?.albumId ?? ""}`,
      ),
    );
  }

  const artistName = labels[0];
  return finalizeMix(
    {
      id: "deep-cuts",
      title: "Deep Cuts",
      subtitle: artistName
        ? `Hidden gems from ${artistName} and more`
        : "Lesser-played tracks from your favorites",
      gradient: gradientFromSeed(`${ctx.daySeed}:deep-cuts`),
    },
    pool,
    `${ctx.daySeed}:deep-cuts:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(labels),
      policy: { familiarRatio: 0.15, discoverRatio: 0.25, maxPerArtist: 4 },
    },
  );
}

async function buildReleaseRadarMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
): Promise<GeneratedMix | null> {
  const favoriteArtists = topArtistLabels(ctx, 6);
  if (favoriteArtists.length === 0) return null;

  const newestByArtist = await Promise.all(
    favoriteArtists.map((label) =>
      fetchers.searchArtistAlbums(label, 8).then((albums) =>
        albums
          .filter(
            (album) => normalizeMixKey(album.artist) === normalizeMixKey(label),
          )
          .sort((a, b) => (b.year ?? 0) - (a.year ?? 0))
          .slice(0, 2),
      ),
    ),
  );

  const matches = dedupeAlbums(newestByArtist.flat()).slice(0, 6);
  if (matches.length === 0) return null;

  const albumSongLists = await Promise.all(
    matches.map((album) => fetchers.getAlbumSongs(album.id)),
  );

  const pool = interleaveByArtist(albumSongLists.flat().slice(0, 36));
  const headline = matches[0]?.artist ?? "your artists";

  return finalizeMix(
    {
      id: "release-radar",
      title: "Release Radar",
      subtitle: `Fresh albums from ${headline}`,
      coverArtId: matches[0]?.coverArt,
      gradient: gradientFromSeed(`${ctx.daySeed}:release-radar`),
    },
    pool,
    `${ctx.daySeed}:release-radar:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(favoriteArtists),
      policy: { familiarRatio: 0.35, discoverRatio: 0.4, maxPerAlbum: 3 },
    },
  );
}

function buildYearRewindMix(
  ctx: MixBuildContext,
  state: MixBuildState,
): GeneratedMix | null {
  const year = new Date().getFullYear();
  const yearStart = new Date(year, 0, 1).getTime();
  const yearEnd = new Date(year + 1, 0, 1).getTime();

  const inYear = historyByListenWeight(ctx.history)
    .filter((entry) => {
      const playedAt = new Date(entry.lastPlayedAt).getTime();
      return playedAt >= yearStart && playedAt < yearEnd;
    })
    .sort((a, b) => b.playCount - a.playCount);

  const minTracks = ctx.settings.minTracksPerMix;
  if (inYear.length < minTracks) return null;

  const tracks = inYear.slice(0, minTracks + 10).map(ctx.entryToSong);
  const topArtist = inYear[0]?.artistName;

  return finalizeMix(
    {
      id: "year-rewind",
      title: `${year} Rewind`,
      subtitle: topArtist
        ? `Your ${year} soundtrack with ${topArtist} and more`
        : `Your most-played tracks from ${year}`,
      gradient: gradientFromSeed(`${ctx.daySeed}:year-rewind:${year}`),
    },
    dedupeTracks(tracks),
    `${ctx.daySeed}:year-rewind:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(
        inYear.slice(0, 8).map((entry) => entry.artistName),
      ),
      policy: {
        familiarRatio: 0.85,
        discoverRatio: 0.1,
        penalizeRecent: false,
      },
    },
  );
}

function buildThrowbackMix(
  ctx: MixBuildContext,
  state: MixBuildState,
): GeneratedMix | null {
  const cutoff = Date.now() - ctx.settings.throwbackDays * 86_400_000;
  const nostalgic = historyByListenWeight(ctx.history)
    .filter(
      (entry) =>
        new Date(entry.lastPlayedAt).getTime() < cutoff &&
        !ctx.skippedTrackIds.has(entry.trackId),
    )
    .sort((a, b) => {
      const age =
        new Date(a.lastPlayedAt).getTime() - new Date(b.lastPlayedAt).getTime();
      if (age !== 0) return age;
      return b.playCount - a.playCount;
    })
    .map(ctx.entryToSong);

  const minTracks = ctx.settings.minTracksPerMix;
  if (nostalgic.length < minTracks) return null;

  return finalizeMix(
    {
      id: "throwback",
      title: "Throwback Mix",
      subtitle: "Tracks you have not heard in a while",
      gradient: gradientFromSeed(`${ctx.daySeed}:throwback`),
    },
    dedupeTracks(nostalgic),
    `${ctx.daySeed}:throwback:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(
        nostalgic.slice(0, 10).map((track) => track.artist ?? ""),
      ),
      policy: {
        familiarRatio: 0.75,
        discoverRatio: 0.1,
        penalizeRecent: false,
      },
    },
  );
}

async function buildTrendingMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
): Promise<GeneratedMix | null> {
  const userAlbumIds = new Set(
    (ctx.stats?.topAlbums ?? []).slice(0, 4).map((album) => album.key),
  );
  const blendedAlbums = dedupeAlbums([
    ...(ctx.stats?.topAlbums ?? []).slice(0, 4).map(
      (entry) =>
        ctx.frequentAlbums.find((album) => album.id === entry.key) ?? {
          id: entry.key,
          name: entry.label,
        },
    ),
    ...ctx.frequentAlbums.slice(0, 4),
  ]).slice(0, 6);

  if (blendedAlbums.length === 0) return null;

  const albumSongLists = await Promise.all(
    blendedAlbums.map((album) => fetchers.getAlbumSongs(album.id)),
  );

  const pool: SubsonicSong[] = [];
  for (const songs of albumSongLists) {
    const preferred = songs.filter((song) =>
      userAlbumIds.has(song.albumId ?? ""),
    );
    pool.push(...(preferred.length > 0 ? preferred : songs).slice(0, 5));
  }

  const subtitle =
    userAlbumIds.size > 0
      ? "Your albums and popular picks on your server"
      : "Popular on your server right now";

  return finalizeMix(
    {
      id: "popular",
      title: "Trending Mix",
      subtitle,
      coverArtId: blendedAlbums[0]?.coverArt,
      gradient: gradientFromSeed(`${ctx.daySeed}:popular`),
    },
    pool,
    `${ctx.daySeed}:popular:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(topArtistLabels(ctx, 6)),
      policy: { familiarRatio: 0.65, discoverRatio: 0.2, maxPerAlbum: 3 },
    },
  );
}

async function buildYourArtistsMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
): Promise<GeneratedMix | null> {
  const labels = topArtistLabels(ctx, 12).slice(4, 10);
  if (labels.length === 0) return null;

  const perArtist = Math.ceil(ctx.settings.minTracksPerMix / labels.length) + 1;
  const results = await Promise.all(
    labels.map((label) => fetchers.searchArtistSongs(label, perArtist + 4)),
  );

  const pool: SubsonicSong[] = [];
  for (const [i, label] of labels.entries()) {
    pool.push(
      ...results[i]
        .filter(
          (song) => normalizeMixKey(song.artist) === normalizeMixKey(label),
        )
        .slice(0, perArtist),
    );
  }

  return finalizeMix(
    {
      id: "your-artists",
      title: "Your Artists Mix",
      subtitle: `Featuring ${labels.slice(0, 3).join(", ")}`,
      gradient: gradientFromSeed(`${ctx.daySeed}:your-artists`),
    },
    pool,
    `${ctx.daySeed}:your-artists:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(labels),
      policy: { familiarRatio: 0.5, discoverRatio: 0.25, maxPerArtist: 4 },
    },
  );
}

async function buildForYouMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
  trackCount: 50 | 100,
): Promise<GeneratedMix | null> {
  const id = trackCount === 50 ? "for-you-50" : "for-you-100";
  const labels = topArtistLabels(ctx, 8);
  const seed = seedFromArtists(
    labels,
    topAffinityKeys(ctx.tasteProfile.genreAffinity, 8),
    rankedDecadeKeys(ctx.tasteProfile.decadeAffinity, 4),
  );

  const seedIds = weightedSimilarSeeds(
    ctx.stats,
    ctx.history,
    `${ctx.daySeed}:${id}:seeds`,
    5,
  );

  const [similar, artistSongGroups, starred, randomAlbums] = await Promise.all([
    similarSongsForSeeds(fetchers, seedIds, 14),
    Promise.all(labels.map((label) => fetchers.searchArtistSongs(label, 12))),
    fetchers.getStarredSongs().catch(() => [] as SubsonicSong[]),
    fetchers.getRandomAlbums(16),
  ]);

  let pool = dedupeTracks([
    ...similar,
    ...artistSongGroups.flat(),
    ...starred,
    ...historyByListenWeight(ctx.history).slice(0, 24).map(ctx.entryToSong),
  ]).filter((track) => !ctx.skippedTrackIds.has(track.id));

  if (pool.length < trackCount + 8) {
    const randomSongs = await fetchers.getRandomSongs(Math.max(trackCount, 40));
    pool = dedupeTracks([...pool, ...randomSongs]).filter(
      (track) => !ctx.skippedTrackIds.has(track.id),
    );
  }

  const albumCover = pickAlbumCover(
    randomAlbums,
    `${ctx.daySeed}:${id}:album-cover`,
  );
  const topName = labels[0];
  return finalizeMix(
    {
      id,
      title: trackCount === 50 ? "For You Mix · 50" : "For You Mix · 100",
      subtitle: topName
        ? `Picks around ${topName} and more from your library`
        : "Picks from your library",
      coverArtId: albumCover,
      gradient: gradientFromSeed(`${ctx.daySeed}:${id}`),
    },
    pool,
    `${ctx.daySeed}:${id}:flow`,
    ctx,
    state,
    {
      maxTracks: trackCount,
      skipCrossMixDedup: true,
      seed,
      policy: {
        familiarRatio: 0.48,
        discoverRatio: 0.35,
        maxPerArtist: 4,
        maxPerAlbum: 2,
        exploreBonus: 0.55,
      },
    },
  );
}

async function buildOverlookedMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
): Promise<GeneratedMix | null> {
  const [randomSongs, randomAlbums] = await Promise.all([
    fetchers.getRandomSongs(80),
    fetchers.getRandomAlbums(16),
  ]);
  const overlooked = randomSongs.filter((track) => {
    const plays = ctx.playCountByTrack.get(track.id) ?? 0;
    return plays === 0 && !ctx.skippedTrackIds.has(track.id);
  });
  if (overlooked.length < ctx.settings.minTracksPerMix) return null;

  return finalizeMix(
    {
      id: "overlooked",
      title: "Overlooked",
      subtitle: "Tracks in your library you have not played yet",
      coverArtId: pickAlbumCover(
        randomAlbums,
        `${ctx.daySeed}:overlooked:album-cover`,
      ),
      gradient: gradientFromSeed(`${ctx.daySeed}:overlooked`),
    },
    overlooked,
    `${ctx.daySeed}:overlooked:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(
        topArtistLabels(ctx, 8),
        topAffinityKeys(ctx.tasteProfile.genreAffinity, 6),
      ),
      policy: {
        familiarRatio: 0.1,
        discoverRatio: 0.7,
        exploreBonus: 0.85,
        penalizeRecent: false,
      },
    },
  );
}

async function buildFavoritesMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
): Promise<GeneratedMix | null> {
  const starred = await fetchers.getStarredSongs();
  const highPlay = historyByListenWeight(ctx.history)
    .filter(
      (entry) =>
        entry.playCount >= 3 && !ctx.skippedTrackIds.has(entry.trackId),
    )
    .slice(0, 40)
    .map(ctx.entryToSong);
  const pool = dedupeTracks([...starred, ...highPlay]);
  if (pool.length < ctx.settings.minTracksPerMix) return null;

  return finalizeMix(
    {
      id: "favorites-mix",
      title: "Favorites Mix",
      subtitle: "Starred tracks and heavy hitters",
      coverArtId: pool[0]?.coverArt,
      gradient: gradientFromSeed(`${ctx.daySeed}:favorites-mix`),
    },
    pool,
    `${ctx.daySeed}:favorites-mix:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(topArtistLabels(ctx, 8)),
      policy: { familiarRatio: 0.85, discoverRatio: 0.1, maxPerArtist: 5 },
    },
  );
}

async function buildDecadeMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
  slot: number,
): Promise<GeneratedMix | null> {
  const decade = rankedDecades(ctx, 6)[slot];
  if (decade === undefined) return null;

  const decadeAlbums = ctx.frequentAlbums.filter(
    (album) =>
      typeof album.year === "number" && decadeFromYear(album.year) === decade,
  );

  const [randomSongs, newest, randomAlbums, frequentSongs] = await Promise.all([
    fetchers.getRandomSongs(80),
    fetchers.getNewestAlbums(16),
    fetchers.getRandomAlbums(24),
    Promise.all(
      decadeAlbums
        .slice(0, 10)
        .map((album) => fetchers.getAlbumSongs(album.id)),
    ),
  ]);

  const candidateAlbums = dedupeAlbums([
    ...decadeAlbums,
    ...newest.filter(
      (album) =>
        typeof album.year === "number" && decadeFromYear(album.year) === decade,
    ),
    ...randomAlbums.filter(
      (album) =>
        typeof album.year === "number" && decadeFromYear(album.year) === decade,
    ),
  ]).slice(0, 14);

  const albumSongLists = await Promise.all(
    candidateAlbums.map((album) => fetchers.getAlbumSongs(album.id)),
  );

  const albumYears = albumYearIndex([
    ...ctx.frequentAlbums,
    ...newest,
    ...randomAlbums,
    ...candidateAlbums,
  ]);

  const pool = dedupeTracks([
    ...frequentSongs.flat(),
    ...albumSongLists.flat(),
    ...randomSongs,
  ]).filter((track) => trackMatchesDecade(track, decade, albumYears));

  if (pool.length < ctx.settings.minTracksPerMix) return null;

  return finalizeMix(
    {
      id: `decade-mix-${slot + 1}`,
      title: `${decade}s Mix`,
      subtitle: `A session rooted in the ${decade}s`,
      coverArtId:
        pickAlbumCover(
          candidateAlbums.length > 0 ? candidateAlbums : randomAlbums,
          `${ctx.daySeed}:decade:${decade}:album-cover`,
        ) ?? pool[0]?.coverArt,
      gradient: gradientFromSeed(`${ctx.daySeed}:decade:${decade}`),
    },
    pool,
    `${ctx.daySeed}:decade:${slot}:flow`,
    ctx,
    state,
    {
      seed: {
        artistKeys: new Set(topArtistLabels(ctx, 8).map(normalizeMixKey)),
        genreKeys: new Set(),
        decades: new Set([decade]),
        artistFocused: false,
      },
      policy: { familiarRatio: 0.5, discoverRatio: 0.3, maxPerArtist: 4 },
    },
  );
}

async function buildFreshFindsMix(
  ctx: MixBuildContext,
  fetchers: MixFetchers,
  state: MixBuildState,
): Promise<GeneratedMix | null> {
  const newest = await fetchers.getNewestAlbums(16);
  if (newest.length === 0) return null;

  const albumSongLists = await Promise.all(
    newest.map((album) => fetchers.getAlbumSongs(album.id)),
  );
  const pool = dedupeTracks(albumSongLists.flat()).filter(
    (track) =>
      !isTopArtistTrack(track, ctx.tasteProfile) &&
      !ctx.skippedTrackIds.has(track.id) &&
      (ctx.playCountByTrack.get(track.id) ?? 0) <= 1,
  );

  if (pool.length < ctx.settings.minTracksPerMix) return null;

  return finalizeMix(
    {
      id: "fresh-finds",
      title: "Fresh Finds",
      subtitle: "New arrivals outside your usual artists",
      coverArtId: newest[0]?.coverArt,
      gradient: gradientFromSeed(`${ctx.daySeed}:fresh-finds`),
    },
    pool,
    `${ctx.daySeed}:fresh-finds:flow`,
    ctx,
    state,
    {
      seed: seedFromArtists(
        [],
        topAffinityKeys(ctx.tasteProfile.genreAffinity, 6),
      ),
      policy: {
        familiarRatio: 0.1,
        discoverRatio: 0.7,
        exploreBonus: 0.8,
        maxPerArtist: 3,
      },
    },
  );
}

export function mixBuilders(
  ctx: MixBuildContext,
  cachedFetchers: MixFetchers,
  state: MixBuildState,
): Array<() => GeneratedMix | null | Promise<GeneratedMix | null>> {
  return [
    () => buildOnRepeatMix(ctx, state),
    () => buildReplayMix(ctx, state),
    () => buildForYouMix(ctx, cachedFetchers, state, 50),
    () => buildForYouMix(ctx, cachedFetchers, state, 100),
    () => buildDailyArtistMix(ctx, cachedFetchers, state, 0),
    () => buildDailyArtistMix(ctx, cachedFetchers, state, 1),
    () => buildDailyArtistMix(ctx, cachedFetchers, state, 2),
    () => buildGenreMix(ctx, cachedFetchers, state, 0),
    () => buildGenreMix(ctx, cachedFetchers, state, 1),
    () => buildGenreMix(ctx, cachedFetchers, state, 2),
    () => buildDiscoverMix(ctx, cachedFetchers, state),
    () => buildDeepCutsMix(ctx, cachedFetchers, state),
    () => buildReleaseRadarMix(ctx, cachedFetchers, state),
    () => buildThrowbackMix(ctx, state),
    () => buildYearRewindMix(ctx, state),
    () => buildTrendingMix(ctx, cachedFetchers, state),
    () => buildYourArtistsMix(ctx, cachedFetchers, state),
    () => buildOverlookedMix(ctx, cachedFetchers, state),
    () => buildFavoritesMix(ctx, cachedFetchers, state),
    () => buildDecadeMix(ctx, cachedFetchers, state, 0),
    () => buildDecadeMix(ctx, cachedFetchers, state, 1),
    () => buildDecadeMix(ctx, cachedFetchers, state, 2),
    () => buildFreshFindsMix(ctx, cachedFetchers, state),
  ];
}
