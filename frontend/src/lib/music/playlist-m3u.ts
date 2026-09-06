// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicConfig, SubsonicSong } from "$lib/subsonic";
import { streamUrl } from "$lib/subsonic/urls";
import { isLocalMusicId } from "$lib/music/library-adapter";
import { localStreamUrl } from "$lib/local-music/api";
import { sanitizeFilename, downloadTextFile } from "$lib/utils/download";

export interface M3UEntry {
  durationSeconds: number;
  title: string;
  artist?: string;
  url: string;
  trackId?: string;
}

export interface M3UPlaylist {
  name?: string;
  entries: M3UEntry[];
}

const EXTINF = /^#EXTINF:(-?\d+)(?:\s+(.+?))?\s*(?:,(.*))?$/i;

function parseExtInfLine(line: string): {
  duration: number;
  title: string;
  artist?: string;
} | null {
  const match = line.match(EXTINF);
  if (!match) return null;

  const duration = Number(match[1]);
  const display = (match[3] ?? match[2] ?? "").trim();
  if (!display) {
    return {
      duration: Number.isFinite(duration) ? duration : -1,
      title: "Unknown",
    };
  }

  const dash = display.match(/^(.+?)\s+-\s+(.+)$/);
  if (dash) {
    return {
      duration: Number.isFinite(duration) ? duration : -1,
      artist: dash[1].trim(),
      title: dash[2].trim(),
    };
  }

  return {
    duration: Number.isFinite(duration) ? duration : -1,
    title: display,
  };
}

function extractTrackIdFromUrl(url: string): string | null {
  try {
    const parsed = new URL(url, window.location.origin);
    const id =
      parsed.searchParams.get("id") ??
      parsed.searchParams.get("trackId") ??
      parsed.searchParams.get("songId");
    if (id) return id;

    const localMatch = parsed.pathname.match(
      /\/api\/local-music\/tracks\/([^/]+)\/stream/,
    );
    if (localMatch?.[1]) return localMatch[1];

    const downloadMatch = parsed.pathname.match(
      /\/api\/downloads\/([^/]+)\/stream/,
    );
    if (downloadMatch?.[1]) return decodeURIComponent(downloadMatch[1]);
  } catch {
    // not a valid URL
  }
  return null;
}

export function parseM3U(content: string): M3UPlaylist {
  const normalized = content.replace(/\r\n/g, "\n").trim();
  const lines = normalized.split("\n");
  const entries: M3UEntry[] = [];
  let pending: { duration: number; title: string; artist?: string } | null =
    null;
  let name: string | undefined;

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line || line === "#EXTM3U") continue;

    if (line.startsWith("#PLAYLIST:")) {
      name = line.slice("#PLAYLIST:".length).trim() || undefined;
      continue;
    }

    if (line.startsWith("#EXTINF:")) {
      pending = parseExtInfLine(line);
      continue;
    }

    if (line.startsWith("#")) continue;

    const meta = pending ?? {
      duration: -1,
      title: line.split("/").pop() ?? "Unknown",
    };
    pending = null;
    const trackId = extractTrackIdFromUrl(line);
    entries.push({
      durationSeconds: meta.duration,
      title: meta.title,
      artist: meta.artist,
      url: line,
      trackId: trackId ?? undefined,
    });
  }

  return { name, entries };
}

export function m3uEntryLabel(entry: M3UEntry): string {
  const artist = entry.artist?.trim();
  const title = entry.title.trim() || "Unknown";
  return artist ? `${artist} - ${title}` : title;
}

function trackStreamLocation(
  config: SubsonicConfig,
  track: SubsonicSong,
): string {
  if (isLocalMusicId(track.id)) {
    return localStreamUrl(track.id);
  }
  return streamUrl(config, track.id);
}

export function buildM3UFromTracks(
  tracks: SubsonicSong[],
  config: SubsonicConfig,
  playlistName?: string,
): string {
  const lines = ["#EXTM3U"];
  if (playlistName?.trim()) {
    lines.push(`#PLAYLIST:${playlistName.trim()}`);
  }

  for (const track of tracks) {
    const duration = track.duration ?? -1;
    const artist = track.artist?.trim();
    const title = track.title?.trim() || "Unknown";
    const label = artist ? `${artist} - ${title}` : title;
    lines.push(`#EXTINF:${duration},${label}`);
    lines.push(trackStreamLocation(config, track));
  }

  return `${lines.join("\n")}\n`;
}

export function exportPlaylistM3U(
  tracks: SubsonicSong[],
  config: SubsonicConfig,
  playlistName: string,
): void {
  const body = buildM3UFromTracks(tracks, config, playlistName);
  const filename = `${sanitizeFilename(playlistName, "playlist")}.m3u`;
  downloadTextFile(filename, body, "audio/x-mpegurl;charset=utf-8");
}

export function normalizeMatchText(value: string): string {
  return value.trim().toLowerCase().replace(/\s+/g, " ");
}

export function matchM3UEntryToSong(
  entry: M3UEntry,
  library: SubsonicSong[],
): SubsonicSong | null {
  if (entry.trackId) {
    const byId = library.find((song) => song.id === entry.trackId);
    if (byId) return byId;
  }

  const entryTitle = normalizeMatchText(entry.title);
  const entryArtist = normalizeMatchText(entry.artist ?? "");
  const entryLabel = normalizeMatchText(m3uEntryLabel(entry));

  for (const song of library) {
    const title = normalizeMatchText(song.title);
    const artist = normalizeMatchText(song.artist ?? "");
    const label = artist ? `${artist} - ${title}` : title;
    if (label === entryLabel) return song;
    if (title === entryTitle && (!entryArtist || artist === entryArtist)) {
      return song;
    }
  }

  return null;
}
