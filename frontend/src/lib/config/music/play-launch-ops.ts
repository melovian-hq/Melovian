// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import {
  createLibraryPoolState,
  type ContinuousMode,
  type LibraryPoolState,
} from "$lib/music/continuous-pool";
import {
  createPersonalRadioState,
  type PersonalRadioState,
} from "$lib/music/personal-radio";
import {
  absoluteMediaUri,
  parseOpenMediaUri,
  titleFromStreamUri,
  trackFromOpenStream,
} from "$lib/music/open-uri";
import {
  truncateTrackList,
  type QueueSettings,
} from "$lib/music/queue-settings";
import type { QueueTrack, SubsonicSong } from "$lib/subsonic";
import { toast } from "$lib/ui/toast.svelte";
import type { PlayerLayout } from "./types";

export interface MusicPlayLaunchContext {
  continuousMode: ContinuousMode;
  libraryPool: LibraryPoolState;
  personalRadio: PersonalRadioState;
  queueSettings: QueueSettings;
  queue: QueueTrack[];
  queueIndex: number;
  shuffleHistory: number[];
  shuffleUpcoming: number[];
  shuffle: boolean;
  autoplay: boolean;
  repeat: "off" | "all" | "one";
  playerLayout: PlayerLayout;
  resumeOnPlay: boolean;
  connected: boolean;
  library: MusicLibraryAdapter;
  seedShuffleUpcoming(): void;
  persistPlaybackState(): void;
  requestPlayCurrent(): void;
  resolveTracksForRestore(trackIds: string[]): Promise<SubsonicSong[]>;
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
  bootstrapOfflinePlayback(): Promise<void>;
  isDownloaded(trackId: string): boolean;
  resolveOfflineTrack(trackId: string): Promise<SubsonicSong | null>;
  connect(options?: {
    quiet?: boolean;
    authEnabled?: boolean;
    force?: boolean;
  }): Promise<boolean>;
  playTrackById(trackId: string): Promise<void>;
}

export function playTracks(
  ctx: MusicPlayLaunchContext,
  tracks: SubsonicSong[],
  startIndex = 0,
  resume = false,
  continuousMode: ContinuousMode | boolean = "off",
  options: {
    feedback?: "play" | "shuffle" | "queue" | false;
    preservePersonalRadio?: boolean;
    preserveLibraryPool?: boolean;
  } = {},
) {
  ctx.continuousMode =
    typeof continuousMode === "boolean"
      ? continuousMode
        ? "random"
        : "off"
      : continuousMode;
  if (ctx.continuousMode === "library" && !options.preserveLibraryPool) {
    ctx.libraryPool = createLibraryPoolState();
  }
  if (ctx.continuousMode === "personal" && !options.preservePersonalRadio) {
    ctx.personalRadio = createPersonalRadioState();
  }
  const limited = truncateTrackList(tracks, ctx.queueSettings.maxQueueSize);
  if (limited.length === 0) return;
  ctx.queue = limited.map((t) => ({ ...t }));
  ctx.queueIndex = Math.max(0, Math.min(startIndex, limited.length - 1));
  ctx.shuffleHistory = [];
  if (ctx.shuffle) {
    ctx.seedShuffleUpcoming();
  } else {
    ctx.shuffleUpcoming = [];
    ctx.shuffleHistory = [];
  }
  if (ctx.playerLayout === "dismissed") ctx.playerLayout = "full";
  ctx.resumeOnPlay = resume;
  ctx.persistPlaybackState();
  ctx.requestPlayCurrent();
  const feedback =
    options.feedback ??
    (limited.length > 1 ? (ctx.shuffle ? "shuffle" : "play") : false);
  if (feedback === "shuffle") {
    toast.success(`Shuffling ${limited.length} tracks`);
  } else if (feedback === "play") {
    toast.success(`Playing ${limited.length} tracks`);
  }
}

export async function applyRemoteQueue(
  ctx: MusicPlayLaunchContext,
  trackIds: string[],
  startIndex = 0,
) {
  const tracks = await ctx.resolveTracksForRestore(trackIds);
  if (tracks.length === 0) return;
  ctx.playTracks(tracks, startIndex);
}

export function playAlbum(
  ctx: MusicPlayLaunchContext,
  songs: SubsonicSong[],
  startIndex = 0,
) {
  ctx.playTracks(songs, startIndex);
}

export async function playTrackById(
  ctx: MusicPlayLaunchContext,
  trackId: string,
) {
  if (!ctx.connected) {
    await ctx.bootstrapOfflinePlayback();
    if (ctx.isDownloaded(trackId)) {
      const song = await ctx.resolveOfflineTrack(trackId);
      if (song) {
        ctx.playTracks([song], 0);
        return;
      }
    }
    const ok = await ctx.connect({ quiet: true });
    if (!ok) throw new Error("Music server unavailable");
  }

  const song = await ctx.library.getSong(trackId);
  if (!song) throw new Error("Track not found");
  ctx.playTracks([song], 0);
}

export async function playOpenUri(ctx: MusicPlayLaunchContext, uri: string) {
  const parsed = parseOpenMediaUri(uri);
  if (!parsed) {
    toast.error("Unsupported media URI");
    return;
  }

  if (!ctx.connected) {
    await ctx.bootstrapOfflinePlayback();
    if (parsed.kind === "trackId" && ctx.isDownloaded(parsed.trackId)) {
      try {
        await ctx.playTrackById(parsed.trackId);
        return;
      } catch {
        /* fall through to stream fallback below */
      }
    }
    const ok = await ctx.connect({ quiet: true });
    if (!ok) {
      toast.error("Music server unavailable");
      return;
    }
  }

  ctx.shuffle = false;
  ctx.autoplay = false;
  ctx.repeat = "off";

  if (parsed.kind === "trackId") {
    try {
      await ctx.playTrackById(parsed.trackId);
      return;
    } catch {
      const fallback = absoluteMediaUri(uri);
      ctx.playTracks(
        [trackFromOpenStream(fallback, titleFromStreamUri(fallback))],
        0,
      );
      return;
    }
  }

  ctx.playTracks([trackFromOpenStream(parsed.streamUrl, parsed.title)], 0);
}

export function createPlayLaunchOps(ctx: MusicPlayLaunchContext) {
  return {
    playTracks: (
      tracks: SubsonicSong[],
      startIndex?: number,
      resume?: boolean,
      continuousMode?: ContinuousMode | boolean,
      options?: {
        feedback?: "play" | "shuffle" | "queue" | false;
        preservePersonalRadio?: boolean;
        preserveLibraryPool?: boolean;
      },
    ) => playTracks(ctx, tracks, startIndex, resume, continuousMode, options),
    applyRemoteQueue: (trackIds: string[], startIndex?: number) =>
      applyRemoteQueue(ctx, trackIds, startIndex),
    playAlbum: (songs: SubsonicSong[], startIndex?: number) =>
      playAlbum(ctx, songs, startIndex),
    playTrackById: (trackId: string) => playTrackById(ctx, trackId),
    playOpenUri: (uri: string) => playOpenUri(ctx, uri),
  };
}
