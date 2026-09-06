// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicSong } from "./types";

export function formatTrackQuality(
  track: Partial<
    Pick<
      SubsonicSong,
      | "suffix"
      | "bitRate"
      | "contentType"
      | "transcoded"
      | "channels"
      | "channelCount"
      | "title"
      | "album"
      | "path"
    >
  >,
): string | null {
  const suffix = track.suffix?.toLowerCase().trim();
  const contentType = track.contentType?.toLowerCase() ?? "";
  const bitRate = track.bitRate ?? 0;

  if (track.transcoded) {
    if (bitRate >= 320) return `MP3 ${bitRate}`;
    if (bitRate > 0) return `MP3 ${bitRate}`;
    return "Transcoded";
  }

  if (suffix === "flac" || contentType.includes("flac")) return "FLAC";
  if (suffix === "alac" || contentType.includes("alac")) return "ALAC";
  if (suffix === "aac" || contentType.includes("aac")) {
    return bitRate > 0 ? `AAC ${bitRate}` : "AAC";
  }
  if (suffix === "ogg" || contentType.includes("ogg")) {
    return bitRate > 0 ? `OGG ${bitRate}` : "OGG";
  }
  if (suffix === "opus" || contentType.includes("opus")) {
    return bitRate > 0 ? `Opus ${bitRate}` : "Opus";
  }
  if (suffix === "eac3" || suffix === "ec3" || contentType.includes("eac3")) {
    return "E-AC-3";
  }
  if (suffix === "ac3" || contentType.includes("ac3")) return "AC-3";
  if (
    suffix === "thd" ||
    suffix === "truehd" ||
    contentType.includes("true-hd") ||
    contentType.includes("truehd")
  ) {
    return "TrueHD";
  }
  if (suffix === "dts" || suffix === "dtshd" || contentType.includes("dts")) {
    return "DTS";
  }
  if (suffix === "mp3" || contentType.includes("mpeg")) {
    if (bitRate >= 320) return "MP3 320";
    if (bitRate >= 256) return "MP3 256";
    if (bitRate >= 192) return "MP3 192";
    if (bitRate > 0) return `MP3 ${bitRate}`;
    return "MP3";
  }
  if (suffix === "m4a" || contentType.includes("mp4")) {
    return bitRate > 0 ? `AAC ${bitRate}` : "M4A";
  }
  if (suffix === "wav" || contentType.includes("wav")) return "WAV";
  if (suffix) return suffix.toUpperCase();
  if (bitRate > 0) return `${bitRate} kbps`;
  return null;
}

export function isLosslessQuality(label: string | null): boolean {
  if (!label) return false;
  return /^(FLAC|ALAC|WAV|TrueHD)/i.test(label);
}
