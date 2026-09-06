// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SlimPersonalMix } from "$lib/music/mix-storage";
import { playlistHref, type PlaylistKind } from "$lib/music/playlist-display";
import { isUnknownArtistName } from "$lib/music/unknown-metadata";
import type {
  ListenEntry,
  MusicPlaylist,
  ServerPlaylist,
  SubsonicAlbum,
  SubsonicArtist,
} from "$lib/subsonic/types";

export type HomeShortcutKind =
  "favorites" | "mix" | "album" | "playlist" | "artist" | "history";

export interface HomeShortcut {
  id: string;
  href: string;
  title: string;
  subtitle?: string;
  coverArtId?: string;
  seed: string;
  kind: HomeShortcutKind;
}

export interface HomePlaylistRef {
  id: string;
  playlistId: string;
  name: string;
  kind: PlaylistKind;
  href: string;
  coverArtId?: string;
  coverArtIds?: string[];
  trackCount: number;
}

export interface BecauseYouListened {
  artist: string;
  albums: SubsonicAlbum[];
}

export interface HomeFeedInput {
  resumeTracks: readonly ListenEntry[];
  listenHistory: readonly ListenEntry[];
  recentAlbums: readonly SubsonicAlbum[];
  frequentAlbums: readonly SubsonicAlbum[];
  recommendations: readonly SubsonicAlbum[];
  favoriteAlbums: readonly SubsonicAlbum[];
  favoriteArtists: readonly SubsonicArtist[];
  favoriteTrackCount: number;
  personalMixes: readonly SlimPersonalMix[];
  playlists: readonly MusicPlaylist[];
  serverPlaylists: readonly ServerPlaylist[];
}

const UNKNOWN_ARTISTS = new Set([
  "",
  "unknown",
  "unknown artist",
  "various artists",
]);

export function greetingForHour(hour: number): string {
  if (hour >= 5 && hour < 12) return "Good morning";
  if (hour >= 12 && hour < 18) return "Good afternoon";
  return "Good evening";
}

export function greetingNow(now = new Date()): string {
  return greetingForHour(now.getHours());
}

export function normalizeArtistName(name: string | undefined): string {
  return name?.trim().toLowerCase() ?? "";
}

export function isNamedArtist(name: string | undefined): boolean {
  if (isUnknownArtistName(name)) return false;
  const key = normalizeArtistName(name);
  return key.length > 0 && !UNKNOWN_ARTISTS.has(key);
}

export function albumFromListen(entry: ListenEntry): SubsonicAlbum | null {
  const id = entry.albumId?.trim();
  if (!id) return null;
  return {
    id,
    name: entry.albumTitle?.trim() || "Unknown album",
    artist: entry.artistName,
    coverArt: entry.coverArtId || undefined,
  };
}

export function uniqueAlbumsById(
  albums: readonly SubsonicAlbum[],
  limit = 12,
): SubsonicAlbum[] {
  const seen = new Set<string>();
  const unique: SubsonicAlbum[] = [];
  for (const album of albums) {
    if (!album.id || seen.has(album.id)) continue;
    seen.add(album.id);
    unique.push(album);
    if (unique.length >= limit) break;
  }
  return unique;
}

export function jumpBackInAlbums(
  resume: readonly ListenEntry[],
  history: readonly ListenEntry[],
  limit = 12,
): SubsonicAlbum[] {
  const ordered = [...resume, ...history];
  const albums: SubsonicAlbum[] = [];
  for (const entry of ordered) {
    const album = albumFromListen(entry);
    if (album) albums.push(album);
  }
  return uniqueAlbumsById(albums, limit);
}

export function albumsMatchingArtist(
  albums: readonly SubsonicAlbum[],
  artist: string,
  limit = 12,
): SubsonicAlbum[] {
  const key = normalizeArtistName(artist);
  if (!key) return [];
  return uniqueAlbumsById(
    albums.filter((album) => normalizeArtistName(album.artist) === key),
    limit,
  );
}

export function becauseYouListened(
  history: readonly ListenEntry[],
  albums: readonly SubsonicAlbum[],
  minAlbums = 2,
): BecauseYouListened | null {
  for (const entry of history) {
    if (!isNamedArtist(entry.artistName)) continue;
    const matched = albumsMatchingArtist(albums, entry.artistName);
    if (matched.length >= minAlbums) {
      return { artist: entry.artistName.trim(), albums: matched };
    }
  }
  return null;
}

export function takeUnusedAlbums(
  albums: readonly SubsonicAlbum[],
  usedIds: Set<string>,
  limit = 12,
): SubsonicAlbum[] {
  const taken: SubsonicAlbum[] = [];
  for (const album of albums) {
    if (!album.id || usedIds.has(album.id)) continue;
    usedIds.add(album.id);
    taken.push(album);
    if (taken.length >= limit) break;
  }
  return taken;
}

export function homePlaylists(
  local: readonly MusicPlaylist[],
  server: readonly ServerPlaylist[],
  limit = 12,
): HomePlaylistRef[] {
  const items: HomePlaylistRef[] = [];
  for (const playlist of local) {
    items.push({
      id: `local:${playlist.id}`,
      playlistId: playlist.id,
      name: playlist.name,
      kind: "local",
      href: playlistHref("local", playlist.id),
      coverArtIds: playlist.coverArtIds,
      trackCount: playlist.trackCount,
    });
    if (items.length >= limit) return items;
  }
  for (const playlist of server) {
    items.push({
      id: `server:${playlist.id}`,
      playlistId: playlist.id,
      name: playlist.name,
      kind: "server",
      href: playlistHref("server", playlist.id),
      coverArtId: playlist.coverArt,
      trackCount: playlist.songCount ?? 0,
    });
    if (items.length >= limit) return items;
  }
  return items;
}

export function buildHomeShortcuts(
  input: HomeFeedInput,
  limit = 8,
): HomeShortcut[] {
  const shortcuts: HomeShortcut[] = [];
  const used = new Set<string>();

  const push = (item: HomeShortcut) => {
    if (shortcuts.length >= limit || used.has(item.id)) return;
    used.add(item.id);
    shortcuts.push(item);
  };

  if (input.favoriteTrackCount > 0) {
    push({
      id: "favorites",
      href: "/music/favorites",
      title: "Liked tracks",
      subtitle: `${input.favoriteTrackCount} saved`,
      seed: "favorites",
      kind: "favorites",
    });
  }

  const firstMix = input.personalMixes[0];
  if (firstMix) {
    push({
      id: `mix:${firstMix.id}`,
      href: `/music/mix/${firstMix.id}`,
      title: firstMix.title,
      subtitle: firstMix.subtitle,
      coverArtId: firstMix.coverArtId,
      seed: firstMix.id,
      kind: "mix",
    });
  }

  const recentAlbum = jumpBackInAlbums(
    input.resumeTracks,
    input.listenHistory,
    1,
  )[0];
  if (recentAlbum) {
    push({
      id: `album:${recentAlbum.id}`,
      href: `/music/album/${recentAlbum.id}`,
      title: recentAlbum.name,
      subtitle: recentAlbum.artist,
      coverArtId: recentAlbum.coverArt,
      seed: recentAlbum.id,
      kind: "album",
    });
  }

  const firstPlaylist = homePlaylists(
    input.playlists,
    input.serverPlaylists,
    1,
  )[0];
  if (firstPlaylist) {
    push({
      id: `playlist:${firstPlaylist.id}`,
      href: firstPlaylist.href,
      title: firstPlaylist.name,
      subtitle: `${firstPlaylist.trackCount} tracks`,
      coverArtId: firstPlaylist.coverArtId ?? firstPlaylist.coverArtIds?.[0],
      seed: firstPlaylist.id,
      kind: "playlist",
    });
  }

  const firstArtist = input.favoriteArtists[0];
  if (firstArtist) {
    push({
      id: `artist:${firstArtist.id}`,
      href: `/music/artist/${firstArtist.id}`,
      title: firstArtist.name,
      seed: firstArtist.id,
      coverArtId: firstArtist.coverArt,
      kind: "artist",
    });
  }

  if (input.listenHistory.length > 0 || input.resumeTracks.length > 0) {
    push({
      id: "history",
      href: "/music/history",
      title: "Recently played",
      seed: "history",
      kind: "history",
    });
  }

  const newest = input.recentAlbums[0];
  if (newest) {
    push({
      id: `newest:${newest.id}`,
      href: `/music/album/${newest.id}`,
      title: newest.name,
      subtitle: newest.artist,
      coverArtId: newest.coverArt,
      seed: newest.id,
      kind: "album",
    });
  }

  const secondMix = input.personalMixes[1];
  if (secondMix) {
    push({
      id: `mix:${secondMix.id}`,
      href: `/music/mix/${secondMix.id}`,
      title: secondMix.title,
      subtitle: secondMix.subtitle,
      coverArtId: secondMix.coverArtId,
      seed: secondMix.id,
      kind: "mix",
    });
  }

  return shortcuts;
}

export function homeHasLibraryContent(input: HomeFeedInput): boolean {
  return (
    input.personalMixes.length > 0 ||
    input.recentAlbums.length > 0 ||
    input.frequentAlbums.length > 0 ||
    input.recommendations.length > 0 ||
    input.favoriteAlbums.length > 0 ||
    input.favoriteArtists.length > 0 ||
    input.favoriteTrackCount > 0 ||
    input.resumeTracks.length > 0 ||
    input.listenHistory.length > 0 ||
    input.playlists.length > 0 ||
    input.serverPlaylists.length > 0
  );
}
