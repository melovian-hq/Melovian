// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { music } from "$lib/config/music.svelte";
import * as musicApi from "$lib/music/api";
import type { PlaylistKind } from "$lib/music/playlist-display";
import { toast } from "$lib/ui/toast.svelte";
import type { PlaylistTrack, SubsonicSong } from "$lib/subsonic";

export function playlistTrackToSong(track: PlaylistTrack): SubsonicSong {
  return {
    id: track.trackId,
    title: track.trackTitle,
    artist: track.artistName,
    album: track.albumTitle,
    albumId: track.albumId,
    coverArt: track.coverArtId,
    duration: Math.floor(track.durationMs / 1000),
  };
}

export async function loadAlbumSongs(albumId: string): Promise<SubsonicSong[]> {
  const detail = await music.library.getAlbum(albumId).catch(() => null);
  return detail?.songs ?? [];
}

export async function playAlbumNow(albumId: string): Promise<void> {
  const songs = await loadAlbumSongs(albumId);
  if (songs.length === 0) return;
  music.playAlbum(songs, 0);
}

export async function queueAlbum(albumId: string): Promise<void> {
  const songs = await loadAlbumSongs(albumId);
  if (songs.length === 0) return;
  music.addTracksToQueue(songs);
}

export async function playArtistNow(
  artistId: string,
  shuffle = false,
): Promise<void> {
  const detail = await music.library.getArtist(artistId).catch(() => null);
  if (!detail?.albums.length) return;
  await music.playArtistAlbums(detail.albums, shuffle);
}

export async function queueArtist(artistId: string): Promise<void> {
  const detail = await music.library.getArtist(artistId).catch(() => null);
  if (!detail?.albums.length) return;
  const songs: SubsonicSong[] = [];
  for (const album of detail.albums) {
    songs.push(...(await loadAlbumSongs(album.id)));
  }
  if (songs.length === 0) return;
  music.addTracksToQueue(songs);
}

export async function loadPlaylistSongs(
  kind: PlaylistKind,
  playlistId: string,
): Promise<SubsonicSong[]> {
  if (kind === "local") {
    const playlist = await musicApi.getPlaylist(playlistId);
    return (playlist.tracks ?? []).map(playlistTrackToSong);
  }
  const result = await music.library
    .getServerPlaylist(playlistId)
    .catch(() => null);
  return result?.songs ?? [];
}

export async function playPlaylistNow(
  kind: PlaylistKind,
  playlistId: string,
): Promise<void> {
  const songs = await loadPlaylistSongs(kind, playlistId);
  if (songs.length === 0) return;
  music.playTracks(songs, 0);
}

export async function queuePlaylist(
  kind: PlaylistKind,
  playlistId: string,
): Promise<void> {
  const songs = await loadPlaylistSongs(kind, playlistId);
  if (songs.length === 0) return;
  music.addTracksToQueue(songs);
}

export async function downloadSongsWithToast(
  songs: SubsonicSong[],
  noun: string,
): Promise<void> {
  try {
    const result = await music.downloadTracks(songs);
    if (result.downloaded === 0 && result.failed === 0) {
      toast.success(`${noun} already available offline`);
    } else if (result.failed > 0) {
      toast.error(`Downloaded ${result.downloaded}, ${result.failed} failed`);
    } else {
      toast.success(`Downloaded ${result.downloaded} tracks`);
    }
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : `Failed to download ${noun}`,
    );
  }
}

export async function downloadAlbum(albumId: string): Promise<void> {
  const songs = await loadAlbumSongs(albumId);
  if (songs.length === 0) return;
  await downloadSongsWithToast(songs, "Album");
}

export async function downloadTrack(track: SubsonicSong): Promise<void> {
  await downloadSongsWithToast([track], "Track");
}

export async function saveTrackToDevice(track: SubsonicSong): Promise<void> {
  const { downloadFromUrl, sanitizeFilename } =
    await import("$lib/utils/download");
  const { apiHeaders } = await import("$lib/core/http/client");
  const title = track.title || "track";
  const artist = track.artist || "";
  const filename = sanitizeFilename(artist ? `${artist} - ${title}` : title);
  try {
    await downloadFromUrl(
      musicApi.mediaTrackDownloadUrl(track.id, { title, artist }),
      filename,
      { headers: apiHeaders() },
    );
    toast.success("Saved to device");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Failed to save track");
  }
}

export async function saveAlbumFilesToDevice(albumId: string): Promise<void> {
  const { downloadFromUrl, sanitizeFilename } =
    await import("$lib/utils/download");
  const { apiHeaders } = await import("$lib/core/http/client");
  try {
    await downloadFromUrl(
      musicApi.mediaAlbumDownloadZipUrl(albumId),
      sanitizeFilename("album") + ".zip",
      { headers: apiHeaders() },
    );
    toast.success("Album download started");
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : "Failed to download album",
    );
  }
}

export async function savePlaylistFilesToDevice(
  kind: PlaylistKind,
  playlistId: string,
  name: string,
): Promise<void> {
  const { downloadFromUrl, sanitizeFilename } =
    await import("$lib/utils/download");
  const { apiHeaders } = await import("$lib/core/http/client");
  const url =
    kind === "local"
      ? musicApi.mediaPlaylistDownloadZipUrl(playlistId)
      : musicApi.mediaServerPlaylistDownloadZipUrl(playlistId);
  try {
    await downloadFromUrl(url, sanitizeFilename(name || "playlist") + ".zip", {
      headers: apiHeaders(),
    });
    toast.success("Playlist download started");
  } catch (err) {
    toast.error(
      err instanceof Error ? err.message : "Failed to download playlist",
    );
  }
}
