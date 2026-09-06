// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { isLocalMusicId } from "$lib/music/library-adapter";
import type { SubsonicSong } from "$lib/subsonic";

export interface SmartTrackContext {
  title: string;
  artist: string;
  album: string;
  albumartist: string;
  genre: string;
  language: string;
  comment: string;
  filepath: string;
  codec: string;
  filetype: string;
  year: number | null;
  date: string;
  dateadded: string;
  releasedate: string;
  lastplayed: string;
  bitrate: number | null;
  bitdepth: number | null;
  samplerate: number | null;
  duration: number | null;
  bpm: number | null;
  rating: number | null;
  playcount: number | null;
  loved: boolean | null;
  compilation: boolean | null;
  missing: boolean | null;
}

function readString(raw: Record<string, unknown>, key: string): string {
  const value = raw[key];
  return typeof value === "string" ? value.trim() : "";
}

function readNumber(raw: Record<string, unknown>, key: string): number | null {
  const value = raw[key];
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

function readBool(raw: Record<string, unknown>, key: string): boolean | null {
  const value = raw[key];
  if (typeof value === "boolean") return value;
  if (value === "true") return true;
  if (value === "false") return false;
  return null;
}

export function songToSmartTrack(
  song: SubsonicSong,
  raw: Record<string, unknown> = {},
): SmartTrackContext {
  const suffix = (song.suffix ?? readString(raw, "suffix")).toLowerCase();
  const contentType = (
    song.contentType ?? readString(raw, "contentType")
  ).toLowerCase();
  const codec = suffix || contentType.split("/").pop() || "";
  const filetype = suffix || codec;

  return {
    title: song.title ?? "",
    artist: song.artist ?? "",
    album: song.album ?? "",
    albumartist: readString(raw, "albumArtist") || (song.artist ?? ""),
    genre: readString(raw, "genre"),
    language: readString(raw, "language"),
    comment: readString(raw, "comment"),
    filepath:
      readString(raw, "path") || (isLocalMusicId(song.id) ? song.id : ""),
    codec,
    filetype,
    year: song.year ?? readNumber(raw, "year"),
    date: readString(raw, "date") || readString(raw, "created"),
    dateadded: readString(raw, "created") || readString(raw, "dateAdded"),
    releasedate:
      readString(raw, "releaseDate") || readString(raw, "originalDate"),
    lastplayed: readString(raw, "lastPlayed") || readString(raw, "played"),
    bitrate: song.bitRate ?? readNumber(raw, "bitRate"),
    bitdepth: readNumber(raw, "bitDepth"),
    samplerate: readNumber(raw, "samplingRate"),
    duration: song.duration ?? readNumber(raw, "duration"),
    bpm: readNumber(raw, "bpm"),
    rating: readNumber(raw, "rating") ?? readNumber(raw, "userRating"),
    playcount: readNumber(raw, "playCount"),
    loved: readBool(raw, "starred") ?? readBool(raw, "loved"),
    compilation: readBool(raw, "compilation"),
    missing: readBool(raw, "missing"),
  };
}
