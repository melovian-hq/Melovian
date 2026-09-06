// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import * as musicApi from "$lib/music/api";
import { clearServerPlaylistCoverCache } from "$lib/music/playlist-covers";
import { movePlaylistSong } from "$lib/music/server-playlist-editing";
import {
  createServerPlaylist as createServerPlaylistOnServer,
  deleteServerPlaylist as deleteServerPlaylistOnServer,
  updateServerPlaylist,
  addSongsToServerPlaylist,
  removeSongFromServerPlaylist,
  setServerPlaylistSongOrder,
  type SubsonicClient,
  type SubsonicSong,
  type ServerPlaylist,
  type InternetRadioStation,
} from "$lib/subsonic";
import type { MusicPlaylist } from "$lib/subsonic/types";

export interface MusicPlaylistContext {
  connected: boolean;
  hasSubsonicActive: boolean;
  client: SubsonicClient;
  library: MusicLibraryAdapter;
  playlists: MusicPlaylist[];
  serverPlaylists: ServerPlaylist[];
  internetRadios: InternetRadioStation[];
}

function requireSubsonicClient(ctx: MusicPlaylistContext): SubsonicClient {
  if (!ctx.hasSubsonicActive) {
    throw new Error("Connect to a Subsonic server to edit server playlists");
  }
  return ctx.client;
}

export async function refreshPlaylists(ctx: MusicPlaylistContext) {
  ctx.playlists = await musicApi.listPlaylists();
}

export async function refreshServerPlaylists(ctx: MusicPlaylistContext) {
  if (!ctx.hasSubsonicActive) {
    ctx.serverPlaylists = [];
    clearServerPlaylistCoverCache();
    return;
  }
  ctx.serverPlaylists = await ctx.library.getServerPlaylists().catch(() => []);
  clearServerPlaylistCoverCache();
}

export async function refreshInternetRadios(ctx: MusicPlaylistContext) {
  if (!ctx.connected) {
    ctx.internetRadios = [];
    return;
  }
  ctx.internetRadios = await ctx.library
    .getInternetRadioStations()
    .catch(() => []);
}

export async function fetchServerPlaylist(
  ctx: MusicPlaylistContext,
  id: string,
) {
  return ctx.library.getServerPlaylist(id);
}

export async function createServerPlaylist(
  ctx: MusicPlaylistContext,
  name: string,
  songIds: string[] = [],
) {
  const playlist = await createServerPlaylistOnServer(
    requireSubsonicClient(ctx),
    name,
    songIds,
  );
  await refreshServerPlaylists(ctx);
  return playlist;
}

export async function createSmartPlaylist(
  ctx: MusicPlaylistContext,
  draft: import("$lib/music/smart-playlist/types").SmartPlaylistDraft,
  target: "local" | "server" = "server",
) {
  const support = await musicApi.getSmartPlaylistSupport().catch(() => ({
    supported: true,
    mode: "client" as const,
  }));

  if (target === "server" && support.mode === "navidrome") {
    const { compileSmartPlaylistDraft } =
      await import("$lib/music/smart-playlist/compile");
    const payload = compileSmartPlaylistDraft(draft);
    const playlist = await musicApi.createSmartPlaylist(payload);
    await refreshServerPlaylists(ctx);
    return playlist;
  }

  const { runSmartPlaylist } = await import("$lib/music/smart-playlist/run");
  const tracks = await runSmartPlaylist(draft, ctx.library);
  if (tracks.length === 0) {
    throw new Error("No tracks matched your rules");
  }

  if (target === "server") {
    return createServerPlaylist(
      ctx,
      draft.name.trim(),
      tracks.map((track) => track.id),
    );
  }

  const created = await musicApi.createPlaylist(draft.name.trim(), {
    kind: "smart",
    rulesJson: JSON.stringify(draft),
  });
  const playlistTracks = tracks.map((track, index) => ({
    trackId: track.id,
    trackTitle: track.title,
    artistName: track.artist ?? "",
    albumId: track.albumId ?? "",
    albumTitle: track.album ?? "",
    durationMs: (track.duration ?? 0) * 1000,
    coverArtId: track.coverArt ?? track.albumId ?? track.id,
    position: index,
  }));
  await musicApi.setPlaylistTracks(created.id, playlistTracks);
  await refreshPlaylists(ctx);
  return { id: created.id, name: created.name, kind: "smart" };
}

export async function refreshSmartPlaylist(
  ctx: MusicPlaylistContext,
  playlistId: string,
) {
  const playlist = await musicApi.getPlaylist(playlistId);
  if (playlist.kind !== "smart" || !playlist.rulesJson) {
    throw new Error("Playlist is not a smart playlist");
  }
  let draft: import("$lib/music/smart-playlist/types").SmartPlaylistDraft;
  try {
    draft = JSON.parse(playlist.rulesJson);
  } catch {
    throw new Error("Smart playlist rules are invalid");
  }
  const { runSmartPlaylist } = await import("$lib/music/smart-playlist/run");
  const tracks = await runSmartPlaylist(draft, ctx.library);
  const playlistTracks = tracks.map((track, index) => ({
    trackId: track.id,
    trackTitle: track.title,
    artistName: track.artist ?? "",
    albumId: track.albumId ?? "",
    albumTitle: track.album ?? "",
    durationMs: (track.duration ?? 0) * 1000,
    coverArtId: track.coverArt ?? track.albumId ?? track.id,
    position: index,
  }));
  const updated = await musicApi.setPlaylistTracks(playlistId, playlistTracks);
  await refreshPlaylists(ctx);
  return updated;
}

export async function deleteServerPlaylist(
  ctx: MusicPlaylistContext,
  id: string,
) {
  await deleteServerPlaylistOnServer(requireSubsonicClient(ctx), id);
  await refreshServerPlaylists(ctx);
}

export async function renameServerPlaylist(
  ctx: MusicPlaylistContext,
  id: string,
  name: string,
) {
  const trimmed = name.trim();
  if (!trimmed) {
    throw new Error("Playlist name is required");
  }
  await updateServerPlaylist(requireSubsonicClient(ctx), {
    playlistId: id,
    name: trimmed,
  });
  await refreshServerPlaylists(ctx);
}

export async function addToServerPlaylist(
  ctx: MusicPlaylistContext,
  playlistId: string,
  tracks: SubsonicSong[],
) {
  const songIds = tracks.map((track) => track.id).filter(Boolean);
  if (songIds.length === 0) return;
  await addSongsToServerPlaylist(
    requireSubsonicClient(ctx),
    playlistId,
    songIds,
  );
  await refreshServerPlaylists(ctx);
}

export async function addTracksToServerPlaylist(
  ctx: MusicPlaylistContext,
  playlistId: string,
  tracks: SubsonicSong[],
) {
  await addToServerPlaylist(ctx, playlistId, tracks);
}

export async function removeFromServerPlaylist(
  ctx: MusicPlaylistContext,
  playlistId: string,
  songIndex: number,
) {
  await removeSongFromServerPlaylist(
    requireSubsonicClient(ctx),
    playlistId,
    songIndex,
  );
  await refreshServerPlaylists(ctx);
}

export async function moveServerPlaylistSong(
  ctx: MusicPlaylistContext,
  playlistId: string,
  currentSongIds: readonly string[],
  fromIndex: number,
  toIndex: number,
) {
  const nextSongIds = movePlaylistSong(currentSongIds, fromIndex, toIndex);
  await setServerPlaylistSongOrder(
    requireSubsonicClient(ctx),
    playlistId,
    currentSongIds,
    nextSongIds,
  );
  await refreshServerPlaylists(ctx);
}

export async function addToPlaylist(
  ctx: MusicPlaylistContext,
  playlistId: string,
  track: SubsonicSong,
) {
  await musicApi.addTrackToPlaylist(playlistId, {
    trackId: track.id,
    trackTitle: track.title,
    artistName: track.artist ?? "",
    albumId: track.albumId ?? "",
    albumTitle: track.album ?? "",
    durationMs: (track.duration ?? 0) * 1000,
    coverArtId: track.coverArt ?? track.albumId ?? track.id,
  });
  await refreshPlaylists(ctx);
}

export async function addTracksToPlaylist(
  ctx: MusicPlaylistContext,
  playlistId: string,
  tracks: SubsonicSong[],
) {
  for (const track of tracks) {
    await musicApi.addTrackToPlaylist(playlistId, {
      trackId: track.id,
      trackTitle: track.title,
      artistName: track.artist ?? "",
      albumId: track.albumId ?? "",
      albumTitle: track.album ?? "",
      durationMs: (track.duration ?? 0) * 1000,
      coverArtId: track.coverArt ?? track.albumId ?? track.id,
    });
  }
  await refreshPlaylists(ctx);
}

export async function createPlaylist(ctx: MusicPlaylistContext, name: string) {
  const pl = await musicApi.createPlaylist(name);
  await refreshPlaylists(ctx);
  return pl;
}

export function createPlaylistOps(ctx: MusicPlaylistContext) {
  return {
    refreshPlaylists: () => refreshPlaylists(ctx),
    refreshServerPlaylists: () => refreshServerPlaylists(ctx),
    refreshInternetRadios: () => refreshInternetRadios(ctx),
    fetchServerPlaylist: (id: string) => fetchServerPlaylist(ctx, id),
    createServerPlaylist: (name: string, songIds?: string[]) =>
      createServerPlaylist(ctx, name, songIds),
    createSmartPlaylist: (
      draft: import("$lib/music/smart-playlist/types").SmartPlaylistDraft,
      target?: "local" | "server",
    ) => createSmartPlaylist(ctx, draft, target),
    refreshSmartPlaylist: (playlistId: string) =>
      refreshSmartPlaylist(ctx, playlistId),
    deleteServerPlaylist: (id: string) => deleteServerPlaylist(ctx, id),
    renameServerPlaylist: (id: string, name: string) =>
      renameServerPlaylist(ctx, id, name),
    addToServerPlaylist: (playlistId: string, tracks: SubsonicSong[]) =>
      addToServerPlaylist(ctx, playlistId, tracks),
    addTracksToServerPlaylist: (playlistId: string, tracks: SubsonicSong[]) =>
      addTracksToServerPlaylist(ctx, playlistId, tracks),
    removeFromServerPlaylist: (playlistId: string, songIndex: number) =>
      removeFromServerPlaylist(ctx, playlistId, songIndex),
    moveServerPlaylistSong: (
      playlistId: string,
      currentSongIds: readonly string[],
      fromIndex: number,
      toIndex: number,
    ) =>
      moveServerPlaylistSong(
        ctx,
        playlistId,
        currentSongIds,
        fromIndex,
        toIndex,
      ),
    addToPlaylist: (playlistId: string, track: SubsonicSong) =>
      addToPlaylist(ctx, playlistId, track),
    addTracksToPlaylist: (playlistId: string, tracks: SubsonicSong[]) =>
      addTracksToPlaylist(ctx, playlistId, tracks),
    createPlaylist: (name: string) => createPlaylist(ctx, name),
  };
}
