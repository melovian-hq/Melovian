// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";

/**
 * Immersive audio output modes.
 *
 * auto: keep native channel layout when the OS mixer allows it
 * stereo: force a stereo downmix
 * surround: prefer multichannel PCM (5.1 / 7.1)
 * passthrough: HDMI or S/PDIF bitstream for AC-3, E-AC-3, TrueHD, and DTS
 *   (Dolby Atmos object audio stays intact only when the receiver decodes it)
 * binaural: headphone-friendly stereo with crossfeed
 */
export type ImmersiveAudioMode =
  "auto" | "stereo" | "surround" | "passthrough" | "binaural";

export interface ImmersiveAudioSettings {
  mode: ImmersiveAudioMode;
  /** Prefer exclusive output for passthrough (WASAPI exclusive, and similar). */
  exclusiveOutput: boolean;
  /**
   * When enabled, skip forced transcoding for multichannel or Atmos-capable
   * streams so the original bitstream or channel layout is preserved.
   */
  preserveImmersiveStreams: boolean;
  /**
   * Comma separated list of extra PCM output targets for native mpv
   * playback: device, stdout, fifo:<path>, tcp:<host:port>,
   * tcp-listen:<addr>, unix:<path>, unix-listen:<path>.
   */
  remoteOutputs: string;
}

const STORAGE_KEY = StorageKeys.immersiveAudioSettings;

export const IMMERSIVE_AUDIO_MODE_LABELS: Record<ImmersiveAudioMode, string> = {
  auto: "Auto",
  stereo: "Stereo",
  surround: "Surround PCM",
  passthrough: "Dolby / DTS passthrough",
  binaural: "Headphones (binaural)",
};

export const IMMERSIVE_AUDIO_MODE_HINTS: Record<ImmersiveAudioMode, string> = {
  auto: "Use the system default layout. Multichannel tracks stay multichannel when the output device supports them.",
  stereo: "Always downmix to two channels.",
  surround:
    "Decode to multichannel PCM for 5.1 or 7.1 speakers. Requires native playback.",
  passthrough:
    "Send AC-3, E-AC-3, TrueHD, and DTS bitstreams over HDMI or S/PDIF so an AVR or soundbar can render Dolby Atmos when the stream carries it. Requires native mpv playback. Volume and EQ are limited while passthrough is active.",
  binaural:
    "Downmix for headphones with stereo crossfeed. On the web player this uses a Web Audio crossfeed graph.",
};

const VALID_MODES = new Set<ImmersiveAudioMode>(
  Object.keys(IMMERSIVE_AUDIO_MODE_LABELS) as ImmersiveAudioMode[],
);

export function defaultImmersiveAudioSettings(): ImmersiveAudioSettings {
  return {
    mode: "auto",
    exclusiveOutput: false,
    preserveImmersiveStreams: true,
    remoteOutputs: "",
  };
}

export function mergeImmersiveAudioSettings(
  partial: Partial<ImmersiveAudioSettings> | null | undefined,
): ImmersiveAudioSettings {
  const defaults = defaultImmersiveAudioSettings();
  if (!partial || typeof partial !== "object") return defaults;

  return {
    mode:
      partial.mode && VALID_MODES.has(partial.mode)
        ? partial.mode
        : defaults.mode,
    exclusiveOutput:
      typeof partial.exclusiveOutput === "boolean"
        ? partial.exclusiveOutput
        : defaults.exclusiveOutput,
    preserveImmersiveStreams:
      typeof partial.preserveImmersiveStreams === "boolean"
        ? partial.preserveImmersiveStreams
        : defaults.preserveImmersiveStreams,
    remoteOutputs:
      typeof partial.remoteOutputs === "string"
        ? partial.remoteOutputs
        : defaults.remoteOutputs,
  };
}

export function loadImmersiveAudioSettings(): ImmersiveAudioSettings {
  if (typeof localStorage === "undefined") {
    return defaultImmersiveAudioSettings();
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultImmersiveAudioSettings();
    return mergeImmersiveAudioSettings(
      JSON.parse(raw) as Partial<ImmersiveAudioSettings>,
    );
  } catch {
    return defaultImmersiveAudioSettings();
  }
}

export function saveImmersiveAudioSettings(
  settings: ImmersiveAudioSettings,
): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    /* storage full or unavailable */
  }
}

/** Split a free-form target list into individual specs. */
export function parseRemoteOutputs(raw: string): string[] {
  return raw
    .split(/[\n,]+/)
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0);
}

/** Native backend config payload shared with AudioService. */
export function immersiveAudioToNativeConfig(
  settings: ImmersiveAudioSettings,
): { mode: string; exclusive: boolean; targets: string[] } {
  return {
    mode: settings.mode,
    exclusive: settings.exclusiveOutput && settings.mode === "passthrough",
    targets: parseRemoteOutputs(settings.remoteOutputs),
  };
}
