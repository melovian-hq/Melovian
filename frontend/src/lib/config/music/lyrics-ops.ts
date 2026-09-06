// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import * as musicApi from "$lib/music/api";
import { normalizeParsedLyrics } from "$lib/music/lyrics";
import {
  mergeLyricsSettings,
  type LyricsSettings,
  type LyricsSettingsResponse,
} from "$lib/music/lyrics-settings";
import {
  isInternetRadioTrack,
  type LyricsSearchHit,
  type ParsedLyrics,
  type QueueTrack,
  type SubsonicSong,
} from "$lib/subsonic";
import type {
  FavoriteTrack,
  ListenEntry,
  SubsonicConfig,
} from "$lib/subsonic/types";
import { streamUrl } from "$lib/subsonic/urls";
import { transcribeTrackLyrics } from "$lib/music/lyrics-whisper";
import { logger } from "$lib/core/logger";
import { toast } from "$lib/ui/toast.svelte";
import { tasks } from "$lib/tasks/tasks.svelte";

export interface MusicLyricsContext {
  lyricsOpen: boolean;
  currentTrack: QueueTrack | null;
  currentLyrics: ParsedLyrics | null;
  lyricsLoading: boolean;
  lyricsFetching: boolean;
  /** Bumped on each load/fetch start so stale responses cannot overwrite newer results. */
  lyricsRequestId: number;
  /** Set when load is skipped because a fetch is in flight. */
  pendingLyricsReload: boolean;
  lyricsSettings: LyricsSettingsResponse | null;
  favoriteTracks: FavoriteTrack[];
  listenHistory: ListenEntry[];
  resumeTracks: ListenEntry[];
  randomTracks: SubsonicSong[];
  library: MusicLibraryAdapter;
  config: SubsonicConfig;
  seek(seconds: number): void;
  favoriteToSong(entry: FavoriteTrack): SubsonicSong;
  entryToSong(entry: ListenEntry): SubsonicSong;
}

function beginLyricsRequest(ctx: MusicLyricsContext): number {
  ctx.lyricsRequestId += 1;
  return ctx.lyricsRequestId;
}

function isCurrentLyricsRequest(
  ctx: MusicLyricsContext,
  requestId: number,
  trackId: string,
): boolean {
  return ctx.lyricsRequestId === requestId && ctx.currentTrack?.id === trackId;
}

export function toggleLyricsPanel(ctx: MusicLyricsContext) {
  ctx.lyricsOpen = !ctx.lyricsOpen;
  if (ctx.lyricsOpen) {
    void loadCurrentLyrics(ctx);
  }
}

export async function loadCurrentLyrics(ctx: MusicLyricsContext) {
  const track = ctx.currentTrack;
  if (!track || isInternetRadioTrack(track)) {
    ctx.currentLyrics = null;
    ctx.lyricsLoading = false;
    return;
  }
  if (ctx.lyricsFetching) {
    // Fetch in flight owns the request id. Mark a reload so a track change
    // during fetch still loads the new track afterward.
    ctx.pendingLyricsReload = true;
    return;
  }

  const requestId = beginLyricsRequest(ctx);
  ctx.lyricsLoading = true;
  try {
    const remote = await musicApi.getTrackLyrics(track.id, {
      artist: track.artist,
      title: track.title,
      album: track.album,
      durationSec: track.duration,
    });
    if (!isCurrentLyricsRequest(ctx, requestId, track.id)) return;
    const normalized = normalizeParsedLyrics(remote);
    if (normalized) {
      ctx.currentLyrics = normalized;
      return;
    }
    const fallback = normalizeParsedLyrics(
      await ctx.library.getLyricsForSong(track),
    );
    if (!isCurrentLyricsRequest(ctx, requestId, track.id)) return;
    ctx.currentLyrics = fallback;
  } catch (err) {
    logger.error("Failed to load lyrics", err, { trackId: track.id }, "lyrics");
    try {
      const fallback = normalizeParsedLyrics(
        await ctx.library.getLyricsForSong(track),
      );
      if (!isCurrentLyricsRequest(ctx, requestId, track.id)) return;
      ctx.currentLyrics = fallback;
    } catch (fallbackErr) {
      logger.error(
        "Lyrics library fallback failed",
        fallbackErr,
        { trackId: track.id },
        "lyrics",
      );
      if (!isCurrentLyricsRequest(ctx, requestId, track.id)) return;
      ctx.currentLyrics = null;
    }
  } finally {
    if (ctx.lyricsRequestId === requestId) {
      ctx.lyricsLoading = false;
    }
  }
}

export async function fetchCurrentLyrics(
  ctx: MusicLyricsContext,
): Promise<ParsedLyrics | null> {
  const track = ctx.currentTrack;
  if (!track || isInternetRadioTrack(track)) return null;

  const requestId = beginLyricsRequest(ctx);
  const startedTrackId = track.id;
  ctx.lyricsFetching = true;
  ctx.lyricsLoading = false;
  const lyricsTaskId = tasks.beginLyricsFetch(
    track.title ? `Fetching lyrics · ${track.title}` : "Fetching lyrics",
  );
  try {
    const fetched = normalizeParsedLyrics(
      await musicApi.fetchTrackLyrics(track.id, {
        artist: track.artist,
        title: track.title,
        album: track.album,
        durationSec: track.duration,
      }),
    );
    if (!isCurrentLyricsRequest(ctx, requestId, track.id)) {
      tasks.finishLyricsFetch(lyricsTaskId, true, "Superseded");
      return fetched;
    }
    if (!fetched) {
      toast.error("No lyrics found from enabled providers");
      ctx.currentLyrics = null;
      tasks.finishLyricsFetch(lyricsTaskId, false, "No lyrics found");
      return null;
    }
    ctx.currentLyrics = fetched;
    toast.success(fetched.synced ? "Synced lyrics fetched" : "Lyrics fetched");
    tasks.finishLyricsFetch(
      lyricsTaskId,
      true,
      fetched.synced ? "Synced lyrics ready" : "Lyrics ready",
    );
    return fetched;
  } catch (err) {
    if (!isCurrentLyricsRequest(ctx, requestId, track.id)) {
      tasks.finishLyricsFetch(lyricsTaskId, true, "Superseded");
      return null;
    }
    const message =
      err instanceof Error ? err.message : "Failed to fetch lyrics";
    logger.error("Failed to fetch lyrics", err, { trackId: track.id }, "lyrics");
    toast.error(message);
    tasks.finishLyricsFetch(lyricsTaskId, false, message);
    return null;
  } finally {
    if (ctx.lyricsRequestId === requestId) {
      ctx.lyricsFetching = false;
    }
    ctx.pendingLyricsReload = false;
    // A load may have been skipped while this fetch ran. If the track
    // changed, fetch results belong to the old id and must not stick.
    if (ctx.currentTrack?.id !== startedTrackId) {
      void loadCurrentLyrics(ctx);
    }
  }
}

/**
 * generateWhisperLyrics transcribes the current track's audio with Whisper.
 * A registered client-side WASM engine (lyrics-whisper WASM package) runs in
 * the browser; otherwise the audio is uploaded to the server, which forwards
 * it to the configured whisper.cpp server.
 */
export async function generateWhisperLyrics(
  ctx: MusicLyricsContext,
): Promise<ParsedLyrics | null> {
  const track = ctx.currentTrack;
  if (!track || isInternetRadioTrack(track)) return null;

  const requestId = beginLyricsRequest(ctx);
  const startedTrackId = track.id;
  ctx.lyricsFetching = true;
  ctx.lyricsLoading = false;
  const taskId = tasks.beginLyricsFetch(
    track.title
      ? `Transcribing lyrics · ${track.title}`
      : "Transcribing lyrics",
  );
  try {
    const url = streamUrl(ctx.config, track.id);
    const generated = normalizeParsedLyrics(
      await transcribeTrackLyrics(track.id, url, {
        artist: track.artist,
        title: track.title,
        album: track.album,
        durationSec: track.duration,
      }),
    );
    if (!isCurrentLyricsRequest(ctx, requestId, track.id)) {
      tasks.finishLyricsFetch(taskId, true, "Superseded");
      return generated;
    }
    if (!generated) {
      toast.error("Whisper produced no lyrics");
      tasks.finishLyricsFetch(taskId, false, "No lyrics generated");
      return null;
    }
    ctx.currentLyrics = generated;
    toast.success(
      generated.synced ? "Synced lyrics generated" : "Lyrics generated",
    );
    tasks.finishLyricsFetch(taskId, true, "Lyrics generated");
    return generated;
  } catch (err) {
    if (!isCurrentLyricsRequest(ctx, requestId, track.id)) {
      tasks.finishLyricsFetch(taskId, true, "Superseded");
      return null;
    }
    const message =
      err instanceof Error ? err.message : "Whisper transcription failed";
    logger.error(
      "Failed to generate lyrics with Whisper",
      err,
      { trackId: track.id },
      "lyrics",
    );
    toast.error(message);
    tasks.finishLyricsFetch(taskId, false, message);
    return null;
  } finally {
    if (ctx.lyricsRequestId === requestId) {
      ctx.lyricsFetching = false;
    }
    ctx.pendingLyricsReload = false;
    if (ctx.currentTrack?.id !== startedTrackId) {
      void loadCurrentLyrics(ctx);
    }
  }
}

export async function loadLyricsSettings(ctx: MusicLyricsContext) {
  const remote = await musicApi.getLyricsSettings().catch(() => null);
  if (remote) {
    ctx.lyricsSettings = remote;
  }
}

export async function updateLyricsSettings(
  ctx: MusicLyricsContext,
  settings: LyricsSettings,
) {
  const remote = await musicApi.saveLyricsSettingsRemote({
    ...mergeLyricsSettings(settings),
    resolvedStorageDir: ctx.lyricsSettings?.resolvedStorageDir ?? "",
    defaultStorageDir: ctx.lyricsSettings?.defaultStorageDir ?? "",
    builtinProviders: ctx.lyricsSettings?.builtinProviders ?? [],
    trackCount: ctx.lyricsSettings?.trackCount ?? 0,
    usedBytes: ctx.lyricsSettings?.usedBytes ?? 0,
  });
  ctx.lyricsSettings = remote;
}

export function seekToLyricLine(ctx: MusicLyricsContext, startMs: number) {
  const offsetMs = ctx.currentLyrics?.offsetMs ?? 0;
  const seconds = Math.max(0, (startMs - offsetMs) / 1000);
  ctx.seek(seconds);
}

export async function searchLyrics(
  ctx: MusicLyricsContext,
  query: string,
): Promise<LyricsSearchHit[]> {
  const extraSongs = [
    ...ctx.favoriteTracks.map((entry) => ctx.favoriteToSong(entry)),
    ...ctx.listenHistory.map((entry) => ctx.entryToSong(entry)),
    ...ctx.resumeTracks.map((entry) => ctx.entryToSong(entry)),
    ...ctx.randomTracks,
  ];
  return ctx.library.searchLyricsByText(query, { extraSongs });
}

export function createLyricsOps(ctx: MusicLyricsContext) {
  return {
    toggleLyricsPanel: () => toggleLyricsPanel(ctx),
    loadCurrentLyrics: () => loadCurrentLyrics(ctx),
    fetchCurrentLyrics: () => fetchCurrentLyrics(ctx),
    generateWhisperLyrics: () => generateWhisperLyrics(ctx),
    loadLyricsSettings: () => loadLyricsSettings(ctx),
    updateLyricsSettings: (settings: LyricsSettings) =>
      updateLyricsSettings(ctx, settings),
    seekToLyricLine: (startMs: number) => seekToLyricLine(ctx, startMs),
    searchLyrics: (query: string) => searchLyrics(ctx, query),
  };
}
