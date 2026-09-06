// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { StreamOptions } from "$lib/subsonic/urls";
import {
  shouldPreserveImmersiveStream,
  type ImmersiveTrackInfo,
} from "./immersive-audio";

export type TranscodeFormat = "server-default" | "mp3" | "ogg" | "opus" | "aac";

export const TRANSCODE_FORMAT_LABELS: Record<TranscodeFormat, string> = {
  "server-default": "Server default",
  mp3: "MP3",
  ogg: "OGG Vorbis",
  opus: "Opus",
  aac: "AAC",
};

export const TRANSCODE_BITRATE_OPTIONS = [
  { value: 0, label: "Off" },
  { value: 96, label: "96 kbps" },
  { value: 128, label: "128 kbps" },
  { value: 192, label: "192 kbps" },
  { value: 256, label: "256 kbps" },
  { value: 320, label: "320 kbps" },
] as const;

export interface TranscodingSettings {
  alwaysTranscode: boolean;
  maxBitRate: number;
  format: TranscodeFormat;
}

export interface BuildStreamOptionsContext {
  track?: ImmersiveTrackInfo;
  preserveImmersiveStreams?: boolean;
}

const STORAGE_KEY = "mel-transcoding-settings";

const VALID_FORMATS = new Set<TranscodeFormat>(
  Object.keys(TRANSCODE_FORMAT_LABELS) as TranscodeFormat[],
);

export function defaultTranscodingSettings(): TranscodingSettings {
  return {
    alwaysTranscode: false,
    maxBitRate: 0,
    format: "server-default",
  };
}

export function mergeTranscodingSettings(
  partial: Partial<TranscodingSettings> | null | undefined,
): TranscodingSettings {
  const defaults = defaultTranscodingSettings();
  if (!partial || typeof partial !== "object") return defaults;

  const maxBitRate =
    typeof partial.maxBitRate === "number" && partial.maxBitRate >= 0
      ? partial.maxBitRate
      : defaults.maxBitRate;
  const format =
    partial.format && VALID_FORMATS.has(partial.format)
      ? partial.format
      : defaults.format;

  return {
    alwaysTranscode:
      typeof partial.alwaysTranscode === "boolean"
        ? partial.alwaysTranscode
        : defaults.alwaysTranscode,
    maxBitRate,
    format,
  };
}

export function loadTranscodingSettings(): TranscodingSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultTranscodingSettings();
    return mergeTranscodingSettings(
      JSON.parse(raw) as Partial<TranscodingSettings>,
    );
  } catch {
    return defaultTranscodingSettings();
  }
}

export function saveTranscodingSettings(settings: TranscodingSettings): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
}

export function buildStreamOptions(
  settings: TranscodingSettings,
  forceTranscode = false,
  context?: BuildStreamOptionsContext,
): StreamOptions | undefined {
  const opts: StreamOptions = {};
  let transcode = settings.alwaysTranscode || forceTranscode;

  if (
    transcode &&
    !forceTranscode &&
    context?.track &&
    shouldPreserveImmersiveStream(
      context.track,
      context.preserveImmersiveStreams === true,
    )
  ) {
    transcode = false;
  }

  if (transcode) {
    opts.maxBitRate = settings.maxBitRate > 0 ? settings.maxBitRate : 320;
  } else if (settings.maxBitRate > 0) {
    // Cap bitrate without forcing a format change when preserving immersive.
    if (!(
      context?.track &&
      shouldPreserveImmersiveStream(
        context.track,
        context.preserveImmersiveStreams === true,
      )
    )) {
      opts.maxBitRate = settings.maxBitRate;
    }
  }

  if (settings.format !== "server-default") {
    if (!(
      context?.track &&
      shouldPreserveImmersiveStream(
        context.track,
        context.preserveImmersiveStreams === true,
      )
    )) {
      opts.format = settings.format;
    }
  }

  if (opts.maxBitRate === undefined && opts.format === undefined) {
    return undefined;
  }
  return opts;
}
