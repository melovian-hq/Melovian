// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { formatPlaylistDuration } from "$lib/music/playlist-duration";
import { StorageKeys } from "$lib/brand";
import type { MusicPlaylist, ServerPlaylist } from "$lib/subsonic";

export type PlaylistViewMode = "list" | "grid" | "card";

export type PlaylistKind = "local" | "server";

export type PlaylistItem = MusicPlaylist | ServerPlaylist;

export const PLAYLIST_VIEW_KEY = StorageKeys.playlistView;
export const PLAYLIST_KIND_KEY = StorageKeys.playlistKind;

export function loadPlaylistKind(): PlaylistKind | null {
  try {
    const saved = localStorage.getItem(PLAYLIST_KIND_KEY);
    if (saved === "server" || saved === "local") return saved;
  } catch {
    // ignore
  }
  return null;
}

export function savePlaylistKind(kind: PlaylistKind) {
  try {
    localStorage.setItem(PLAYLIST_KIND_KEY, kind);
  } catch {
    // ignore
  }
}

export function loadPlaylistView(): PlaylistViewMode {
  try {
    const saved = localStorage.getItem(PLAYLIST_VIEW_KEY);
    if (saved === "list" || saved === "grid" || saved === "card") return saved;
  } catch {
    // ignore
  }
  return "grid";
}

export function playlistHref(kind: PlaylistKind, id: string): string {
  return kind === "server"
    ? `/music/server-playlist/${id}`
    : `/music/playlist/${id}`;
}

export function playlistTrackCount(
  playlist: PlaylistItem,
  kind: PlaylistKind,
): number {
  if (kind === "server") {
    return (playlist as ServerPlaylist).songCount ?? 0;
  }
  return (playlist as MusicPlaylist).trackCount ?? 0;
}

export function playlistDurationLabel(
  playlist: PlaylistItem,
  kind: PlaylistKind,
): string | null {
  if (kind === "server") {
    return formatPlaylistDuration(playlist as ServerPlaylist);
  }
  return formatPlaylistDuration(playlist as MusicPlaylist);
}

export function playlistMetaParts(
  playlist: PlaylistItem,
  kind: PlaylistKind,
): string[] {
  const parts: string[] = [];
  const count = playlistTrackCount(playlist, kind);
  parts.push(`${count.toLocaleString()} track${count === 1 ? "" : "s"}`);

  const duration = playlistDurationLabel(playlist, kind);
  if (duration) parts.push(duration);

  if (kind === "server") {
    const server = playlist as ServerPlaylist;
    if (server.owner) parts.push(server.owner);
    if (server.public) parts.push("Public");
  }

  return parts;
}

export function coverStackVariant(
  view: PlaylistViewMode,
): "compact" | "grid" | "card" {
  if (view === "list") return "compact";
  return "card";
}
