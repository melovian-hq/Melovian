// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";

export interface PlaybackSettings {
  crossfadeEnabled: boolean;
  crossfadeDurationSec: number;
  continuePlaybackOnLaunch: boolean;
}

const STORAGE_KEY = StorageKeys.playbackSettings;

export const CROSSFADE_DURATION_OPTIONS = [
  { value: 2, label: "2 seconds" },
  { value: 3, label: "3 seconds" },
  { value: 5, label: "5 seconds" },
  { value: 8, label: "8 seconds" },
  { value: 12, label: "12 seconds" },
] as const;

export function defaultPlaybackSettings(): PlaybackSettings {
  return {
    crossfadeEnabled: false,
    crossfadeDurationSec: 3,
    continuePlaybackOnLaunch: false,
  };
}

function clampDuration(value: unknown, fallback: number): number {
  if (typeof value !== "number" || !Number.isFinite(value)) return fallback;
  return Math.max(1, Math.min(12, Math.round(value)));
}

export function mergePlaybackSettings(
  partial: Partial<PlaybackSettings> | null | undefined,
): PlaybackSettings {
  const defaults = defaultPlaybackSettings();
  if (!partial || typeof partial !== "object") return defaults;

  return {
    crossfadeEnabled: partial.crossfadeEnabled === true,
    crossfadeDurationSec: clampDuration(
      partial.crossfadeDurationSec,
      defaults.crossfadeDurationSec,
    ),
    continuePlaybackOnLaunch:
      typeof partial.continuePlaybackOnLaunch === "boolean"
        ? partial.continuePlaybackOnLaunch
        : defaults.continuePlaybackOnLaunch,
  };
}

export function loadPlaybackSettings(): PlaybackSettings {
  if (typeof localStorage === "undefined") return defaultPlaybackSettings();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultPlaybackSettings();
    return mergePlaybackSettings(JSON.parse(raw) as Partial<PlaybackSettings>);
  } catch {
    return defaultPlaybackSettings();
  }
}

export function savePlaybackSettings(settings: PlaybackSettings): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    /* storage full or unavailable */
  }
}

export function crossfadeActive(
  settings: PlaybackSettings,
  _nativePlayback = false,
): boolean {
  return settings.crossfadeEnabled;
}
