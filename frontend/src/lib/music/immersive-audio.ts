// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SubsonicSong } from "$lib/subsonic/types";

const IMMERSIVE_SUFFIXES = new Set([
  "ac3",
  "eac3",
  "ec3",
  "thd",
  "truehd",
  "mlp",
  "dts",
  "dtshd",
  "dtsma",
  "dca",
]);

const IMMERSIVE_CONTENT_TYPES = [
  "ac3",
  "eac3",
  "ec-3",
  "true-hd",
  "truehd",
  "mlp",
  "dts",
  "vnd.dts",
];

export type ImmersiveTrackInfo = Partial<
  Pick<
    SubsonicSong,
    | "suffix"
    | "contentType"
    | "channels"
    | "channelCount"
    | "title"
    | "album"
    | "path"
  >
>;

function readText(...values: Array<string | undefined>): string {
  return values
    .filter((v): v is string => typeof v === "string" && v.length > 0)
    .join(" ")
    .toLowerCase();
}

/** channelCount returns the best available channel metadata for a track. */
export function trackChannelCount(track: ImmersiveTrackInfo): number {
  if (typeof track.channels === "number" && track.channels > 0) {
    return track.channels;
  }
  if (typeof track.channelCount === "number" && track.channelCount > 0) {
    return track.channelCount;
  }
  return 0;
}

/** isMultichannelTrack reports whether the track has more than two channels. */
export function isMultichannelTrack(track: ImmersiveTrackInfo): boolean {
  return trackChannelCount(track) > 2;
}

/**
 * isAtmosCapableTrack detects Atmos-capable containers from codec metadata and
 * common naming. True Atmos object decoding still requires a licensed decoder
 * in the AVR or soundbar when using passthrough.
 */
export function isAtmosCapableTrack(track: ImmersiveTrackInfo): boolean {
  const blob = readText(
    track.suffix,
    track.contentType,
    track.title,
    track.album,
    track.path,
  );
  if (/\batmos\b/.test(blob)) return true;
  if (/\bdolby\s*atmos\b/.test(blob)) return true;

  const suffix = track.suffix?.toLowerCase().trim() ?? "";
  if (
    suffix === "eac3" ||
    suffix === "ec3" ||
    suffix === "thd" ||
    suffix === "truehd"
  ) {
    return true;
  }

  const contentType = track.contentType?.toLowerCase() ?? "";
  return (
    contentType.includes("eac3") ||
    contentType.includes("ec-3") ||
    contentType.includes("true-hd") ||
    contentType.includes("truehd")
  );
}

/** isImmersiveCodecTrack detects surround or bitstream codecs worth preserving. */
export function isImmersiveCodecTrack(track: ImmersiveTrackInfo): boolean {
  if (isAtmosCapableTrack(track) || isMultichannelTrack(track)) return true;

  const suffix = track.suffix?.toLowerCase().trim() ?? "";
  if (IMMERSIVE_SUFFIXES.has(suffix)) return true;

  const contentType = track.contentType?.toLowerCase() ?? "";
  return IMMERSIVE_CONTENT_TYPES.some((token) => contentType.includes(token));
}

/**
 * shouldPreserveImmersiveStream returns true when forced transcoding should be
 * skipped so multichannel or Atmos bitstreams are not flattened to stereo MP3.
 */
export function shouldPreserveImmersiveStream(
  track: ImmersiveTrackInfo,
  preserveImmersiveStreams: boolean,
): boolean {
  return preserveImmersiveStreams && isImmersiveCodecTrack(track);
}

/** formatImmersiveBadge builds a short UI label for surround or Atmos tracks. */
export function formatImmersiveBadge(track: ImmersiveTrackInfo): string | null {
  if (isAtmosCapableTrack(track)) return "Atmos";
  const channels = trackChannelCount(track);
  if (channels >= 8) return "7.1";
  if (channels >= 6) return "5.1";
  if (channels > 2) return `${channels}ch`;
  if (isImmersiveCodecTrack(track)) return "Surround";
  return null;
}
