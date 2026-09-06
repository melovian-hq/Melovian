// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import {
  CONTINUOUS_REFILL_BATCH,
  continuousRefillCount,
  createLibraryPoolState,
  pullLibraryTracks,
  type ContinuousMode,
  type LibraryPoolState,
} from "$lib/music/continuous-pool";
import {
  buildPersonalProfile,
  createPersonalRadioState,
  isPersonalColdStart,
  seedPersonalRadioTracks,
  type PersonalRadioOptions,
  type PersonalRadioState,
} from "$lib/music/personal-radio";
import type { MixSettings } from "$lib/music/mix-settings";
import type { QueueSettings } from "$lib/music/queue-settings";
import {
  trackFromRadioStation,
  type InternetRadioStation,
  type SubsonicSong,
} from "$lib/subsonic";
import type { ListenEntry, ListenStats } from "$lib/subsonic/types";
import { toast } from "$lib/ui/toast.svelte";

export interface MusicRadioContext {
  continuousBusy: "off" | ContinuousMode | "internet";
  continuousMode: ContinuousMode;
  shuffle: boolean;
  autoplay: boolean;
  repeat: "off" | "all" | "one";
  library: MusicLibraryAdapter;
  libraryPool: LibraryPoolState;
  personalRadio: PersonalRadioState;
  queueSettings: QueueSettings;
  listenHistory: ListenEntry[];
  stats: ListenStats | null;
  mixSettings: MixSettings;
  internetRadios: InternetRadioStation[];
  playTracks(
    tracks: SubsonicSong[],
    startIndex?: number,
    resume?: boolean,
    continuousMode?: ContinuousMode | boolean,
    options?: {
      feedback?: "play" | "shuffle" | "queue" | false;
      preservePersonalRadio?: boolean;
      preserveLibraryPool?: boolean;
    },
  ): void;
  startRandomRadio(count?: number): Promise<void>;
  playInternetRadio(station: InternetRadioStation): void;
  refreshInternetRadios(): Promise<void>;
  persistPlaybackState(): void;
  entryToSong(entry: ListenEntry): SubsonicSong;
  personalRadioOptions(coldStart?: boolean): PersonalRadioOptions;
}

export async function playRandomRadio(ctx: MusicRadioContext, count = 25) {
  if (ctx.continuousBusy !== "off") return;
  ctx.continuousBusy = "random";
  toast.info("Building random radio…");
  try {
    await ctx.startRandomRadio(count);
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : "Failed to start random radio",
    );
  } finally {
    ctx.continuousBusy = "off";
  }
}

export async function startRandomRadio(ctx: MusicRadioContext, count = 25) {
  const tracks = await ctx.library.getRandomSongs(count);
  if (tracks.length === 0) {
    toast.warning("No tracks available for random radio");
    return;
  }
  ctx.shuffle = true;
  ctx.autoplay = true;
  ctx.playTracks(tracks, 0, false, "random");
}

/**
 * Start continuous random playback from a seed set of tracks. The provided
 * tracks become the initial queue and the queue is refilled with more random
 * songs from the library as playback approaches the end.
 */
export function playRandomTracks(
  ctx: MusicRadioContext,
  tracks: SubsonicSong[],
) {
  if (tracks.length === 0) return;
  ctx.shuffle = true;
  ctx.autoplay = true;
  ctx.playTracks(tracks, 0, false, "random");
}

export async function playLibraryShuffle(ctx: MusicRadioContext) {
  if (ctx.continuousBusy !== "off") return;
  ctx.continuousBusy = "library";
  toast.info("Shuffling your library…");
  try {
    ctx.libraryPool = createLibraryPoolState();
    const target = continuousRefillCount(
      0,
      ctx.queueSettings.maxQueueSize,
      "library",
    );
    const batch = Math.max(target, CONTINUOUS_REFILL_BATCH);
    const tracks = await pullLibraryTracks(
      ctx.libraryPool,
      {
        getAlbumList2: (type, size, offset) =>
          ctx.library.getAlbumList2(type, size, offset),
        getAlbumSongs: async (albumId) => {
          const album = await ctx.library.getAlbum(albumId).catch(() => null);
          return album?.songs ?? [];
        },
      },
      batch,
      new Set(),
    );
    if (tracks.length === 0) {
      await ctx.startRandomRadio();
      return;
    }
    ctx.shuffle = true;
    ctx.autoplay = true;
    ctx.playTracks(tracks, 0, false, "library", {
      feedback: "shuffle",
      preserveLibraryPool: true,
    });
    toast.success("Shuffling your library");
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : "Failed to shuffle library",
    );
  } finally {
    ctx.continuousBusy = "off";
  }
}

export async function playPersonalRadio(ctx: MusicRadioContext, count = 25) {
  if (ctx.continuousBusy !== "off") return;
  ctx.continuousBusy = "personal";
  toast.info("Building your personal radio…");
  try {
    ctx.personalRadio = createPersonalRadioState();
    const history = ctx.listenHistory;
    const stats = ctx.stats;
    const coldStart = isPersonalColdStart(
      history,
      stats,
      ctx.mixSettings.personalRadioColdStartPlays,
    );
    if (coldStart) {
      toast.info("Limited history. Blending discovery tracks.");
    }

    const playCountByTrack = new Map<string, number>();
    const listenedMsByTrack = new Map<string, number>();
    for (const entry of history) {
      playCountByTrack.set(entry.trackId, entry.playCount);
      if ((entry.listenedMs ?? 0) > 0) {
        listenedMsByTrack.set(entry.trackId, entry.listenedMs);
      }
    }
    const profile = buildPersonalProfile(
      stats,
      history,
      playCountByTrack,
      new Set(),
      ctx.personalRadio.skipTrackIds,
      listenedMsByTrack,
    );

    const tracks = await seedPersonalRadioTracks(
      ctx.personalRadio,
      {
        getSimilarSongs: (trackId, c) =>
          ctx.library.getSimilarSongs(trackId, c).catch(() => []),
        getRandomSongs: (c) => ctx.library.getRandomSongs(c).catch(() => []),
        searchArtistSongs: async (artist, limit) => {
          const result = await ctx.library.search3(artist, limit).catch(() => ({
            songs: [] as SubsonicSong[],
          }));
          return result.songs.filter(
            (song) => song.artist?.toLowerCase() === artist.toLowerCase(),
          );
        },
      },
      profile,
      history,
      stats,
      count,
      new Set(),
      (entry) => ctx.entryToSong(entry),
      ctx.personalRadioOptions(coldStart),
    );

    if (tracks.length === 0) {
      await ctx.startRandomRadio(count);
      return;
    }

    ctx.shuffle = true;
    ctx.autoplay = true;
    ctx.playTracks(tracks, 0, false, "personal", {
      feedback: false,
      preservePersonalRadio: true,
    });
    toast.success("Starting your personal radio");
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : "Failed to start personal radio",
    );
  } finally {
    ctx.continuousBusy = "off";
  }
}

export function clearContinuousMode(ctx: MusicRadioContext) {
  ctx.continuousMode = "off";
  ctx.persistPlaybackState();
}

export async function playRandomInternetRadio(ctx: MusicRadioContext) {
  if (ctx.continuousBusy !== "off") return;
  ctx.continuousBusy = "internet";
  toast.info("Tuning internet radio…");
  try {
    if (ctx.internetRadios.length === 0) {
      await ctx.refreshInternetRadios();
    }
    if (ctx.internetRadios.length === 0) {
      await ctx.startRandomRadio();
      return;
    }
    const station =
      ctx.internetRadios[Math.floor(Math.random() * ctx.internetRadios.length)];
    ctx.playInternetRadio(station);
    toast.success(`Playing ${station.name}`);
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : "Failed to start internet radio",
    );
  } finally {
    ctx.continuousBusy = "off";
  }
}

export function playInternetRadio(
  ctx: MusicRadioContext,
  station: InternetRadioStation,
) {
  ctx.shuffle = false;
  ctx.autoplay = false;
  ctx.repeat = "off";
  ctx.playTracks([trackFromRadioStation(station)], 0);
}

export function playInternetRadios(
  ctx: MusicRadioContext,
  stations: InternetRadioStation[],
  startIndex = 0,
) {
  ctx.shuffle = false;
  ctx.autoplay = false;
  ctx.repeat = "off";
  ctx.playTracks(
    stations.map((station) => trackFromRadioStation(station)),
    startIndex,
  );
}

export function createRadioOps(ctx: MusicRadioContext) {
  return {
    playRandomRadio: (count?: number) => playRandomRadio(ctx, count),
    startRandomRadio: (count?: number) => startRandomRadio(ctx, count),
    playRandomTracks: (tracks: SubsonicSong[]) => playRandomTracks(ctx, tracks),
    playLibraryShuffle: () => playLibraryShuffle(ctx),
    playPersonalRadio: (count?: number) => playPersonalRadio(ctx, count),
    clearContinuousMode: () => clearContinuousMode(ctx),
    playRandomInternetRadio: () => playRandomInternetRadio(ctx),
    playInternetRadio: (station: InternetRadioStation) =>
      playInternetRadio(ctx, station),
    playInternetRadios: (
      stations: InternetRadioStation[],
      startIndex?: number,
    ) => playInternetRadios(ctx, stations, startIndex),
  };
}
