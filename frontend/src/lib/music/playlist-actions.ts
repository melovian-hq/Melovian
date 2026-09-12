// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { music } from "$lib/config/music.svelte";
import * as musicApi from "$lib/music/api";
import {
  exportPlaylistM3U,
  matchM3UEntryToSong,
  parseM3U,
} from "$lib/music/playlist-m3u";
import type { PlaylistKind } from "$lib/music/playlist-display";
import { toast } from "$lib/ui/toast.svelte";
import { confirmDialog } from "$lib/ui/confirm.svelte";
import type {
  MusicPlaylist,
  ServerPlaylist,
  SubsonicSong,
} from "$lib/subsonic";
import type { SmartPlaylistDraft } from "$lib/music/smart-playlist/types";

export async function createNamedPlaylist(
  kind: PlaylistKind,
  name: string,
): Promise<boolean> {
  try {
    if (kind === "server") {
      await music.createServerPlaylist(name);
      toast.success("Server playlist created");
    } else {
      await music.createPlaylist(name);
      toast.success("Local playlist created");
    }
    return true;
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : "Failed to create playlist",
    );
    return false;
  }
}

export async function createSmartPlaylistForKind(
  draft: SmartPlaylistDraft,
  kind: PlaylistKind,
): Promise<void> {
  await music.createSmartPlaylist(draft, kind);
  toast.success(`Smart playlist "${draft.name.trim()}" created`);
}

export async function deleteLocalPlaylist(id: string): Promise<void> {
  await musicApi.deletePlaylist(id);
  await music.refreshPlaylists();
}

export async function deleteServerPlaylist(
  id: string,
  name: string,
): Promise<void> {
  const ok = await confirmDialog.confirm({
    title: "Delete server playlist",
    message: `Delete server playlist "${name}"? This cannot be undone.`,
    confirmLabel: "Delete playlist",
    danger: true,
  });
  if (!ok) return;
  try {
    await music.deleteServerPlaylist(id);
    toast.success("Server playlist deleted");
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : "Failed to delete server playlist",
    );
  }
}

export async function exportLocalPlaylist(pl: MusicPlaylist): Promise<void> {
  const full = await musicApi.getPlaylist(pl.id);
  const tracks = (full.tracks ?? []).map((track) => ({
    id: track.trackId,
    title: track.trackTitle,
    artist: track.artistName,
    album: track.albumTitle,
    albumId: track.albumId,
    coverArt: track.coverArtId,
    duration: Math.floor(track.durationMs / 1000),
  }));
  exportPlaylistM3U(tracks, music.config, full.name);
  toast.success("Playlist exported");
}

export async function exportServerPlaylist(pl: ServerPlaylist): Promise<void> {
  const result = await music.fetchServerPlaylist(pl.id);
  if (!result) {
    toast.error("Failed to load playlist for export");
    return;
  }
  exportPlaylistM3U(result.songs, music.config, result.playlist.name);
  toast.success("Playlist exported");
}

export async function importM3UPlaylist(
  file: File,
  toServer: boolean,
): Promise<void> {
  try {
    const text = await file.text();
    const parsed = parseM3U(text);
    const playlistName =
      parsed.name?.trim() ||
      file.name.replace(/\.m3u8?$/i, "").trim() ||
      "Imported playlist";

    const matched: SubsonicSong[] = [];
    const seen = new Set<string>();

    for (const entry of parsed.entries) {
      if (entry.trackId) {
        const byId = await music.library
          .getSong(entry.trackId)
          .catch(() => null);
        if (byId && !seen.has(byId.id)) {
          seen.add(byId.id);
          matched.push(byId);
          continue;
        }
      }

      const query = entry.artist
        ? `${entry.artist} ${entry.title}`
        : entry.title;
      const result = await music.library.search3(query, 12).catch(() => ({
        songs: [] as SubsonicSong[],
      }));
      const found = matchM3UEntryToSong(entry, result.songs);
      if (found && !seen.has(found.id)) {
        seen.add(found.id);
        matched.push(found);
      }
    }

    if (matched.length === 0) {
      toast.error("No tracks from this playlist matched your library");
      return;
    }

    if (toServer) {
      await music.createServerPlaylist(
        playlistName,
        matched.map((song) => song.id),
      );
    } else {
      const created = await musicApi.createPlaylist(playlistName);
      for (const song of matched) {
        await musicApi.addTrackToPlaylist(created.id, {
          trackId: song.id,
          trackTitle: song.title,
          artistName: song.artist ?? "",
          albumId: song.albumId ?? "",
          albumTitle: song.album ?? "",
          durationMs: (song.duration ?? 0) * 1000,
          coverArtId: song.coverArt ?? song.albumId ?? song.id,
        });
      }
      await music.refreshPlaylists();
    }

    const skipped = parsed.entries.length - matched.length;
    toast.success(
      skipped > 0
        ? `Imported ${matched.length} tracks (${skipped} unmatched)`
        : `Imported ${matched.length} tracks`,
    );
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : "Failed to import playlist",
    );
  }
}
