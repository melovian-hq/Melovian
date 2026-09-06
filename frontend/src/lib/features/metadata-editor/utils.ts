// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicSong } from "$lib/subsonic";
import type {
  MetadataFilenameSuggestion,
  MetadataLookupMatch,
  MetadataTrack,
  MetadataTrackUpdate,
} from "./types";

export function metadataTrackToSong(track: MetadataTrack): SubsonicSong {
  return {
    id: track.id,
    title: track.title || "Unknown title",
    artist: track.artist,
    album: track.album,
    coverArt: track.coverArt,
    duration: Math.max(0, Math.floor(track.durationMs / 1000)),
  };
}

export interface MetadataFormState {
  title: string;
  artist: string;
  album: string;
  albumArtist: string;
  trackNum: string;
  discNum: string;
  year: string;
  genre: string;
}

export function trackToFormState(track: MetadataTrack): MetadataFormState {
  return {
    title: track.title,
    artist: track.artist,
    album: track.album,
    albumArtist: track.albumArtist,
    trackNum: track.trackNum > 0 ? String(track.trackNum) : "",
    discNum: track.discNum > 0 ? String(track.discNum) : "",
    year: track.year > 0 ? String(track.year) : "",
    genre: track.genre,
  };
}

export function parseOptionalInt(value: string): number | undefined {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  const parsed = Number.parseInt(trimmed, 10);
  return Number.isFinite(parsed) ? parsed : undefined;
}

export function formStateToUpdate(
  form: MetadataFormState,
): MetadataTrackUpdate {
  return {
    title: form.title.trim(),
    artist: form.artist.trim(),
    album: form.album.trim(),
    albumArtist: form.albumArtist.trim(),
    trackNum: parseOptionalInt(form.trackNum),
    discNum: parseOptionalInt(form.discNum),
    year: parseOptionalInt(form.year),
    genre: form.genre.trim(),
  };
}

export function isMetadataFormDirty(
  track: MetadataTrack,
  form: MetadataFormState,
): boolean {
  const current = trackToFormState(track);
  return (Object.keys(current) as (keyof MetadataFormState)[]).some(
    (key) => current[key] !== form[key],
  );
}

export function issueLabel(issue: string): string {
  return issue.replaceAll("-", " ");
}

export function buildLookupQuery(
  track: MetadataTrack,
  override?: string,
): string {
  const custom = override?.trim();
  if (custom) return custom;
  return [track.artist, track.title, track.album]
    .map((part) => part.trim())
    .filter(Boolean)
    .join(" ");
}

export function filenameSuggestionToMatch(
  suggestion: MetadataFilenameSuggestion,
): MetadataLookupMatch {
  return {
    id: "filename",
    source: suggestion.source,
    title: suggestion.title,
    artist: suggestion.artist,
    album: suggestion.album,
    albumArtist: suggestion.albumArtist || suggestion.artist,
    trackNum: suggestion.trackNum,
    year: 0,
    genre: "",
  };
}

export function hasFilenameSuggestion(
  suggestion: MetadataFilenameSuggestion,
): boolean {
  return Boolean(
    suggestion.title.trim() ||
    suggestion.artist.trim() ||
    suggestion.album.trim(),
  );
}

export function selectedTrackIndex(
  tracks: MetadataTrack[],
  selectedId: string | null,
): number {
  if (!selectedId) return -1;
  return tracks.findIndex((track) => track.id === selectedId);
}

export function nextTrackId(
  tracks: MetadataTrack[],
  selectedId: string | null,
  direction: 1 | -1,
): string | null {
  const index = selectedTrackIndex(tracks, selectedId);
  if (index < 0) return tracks[0]?.id ?? null;
  const next = tracks[index + direction];
  return next?.id ?? selectedId;
}
