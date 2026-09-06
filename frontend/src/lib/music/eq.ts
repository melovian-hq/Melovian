// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  defaultBandParams,
  defaultEqSettings,
  EQ_BAND_COUNT,
  EQ_FREQ_MAX,
  EQ_FREQ_MIN,
  EQ_GAIN_MAX,
  EQ_GAIN_MIN,
  EQ_PRESETS,
  EQ_Q_MAX,
  EQ_Q_MIN,
  type EqBandParam,
  type EqSettings,
} from "./eq-data";
import { StorageKeys } from "$lib/brand";

const STORAGE_KEY = StorageKeys.eq;

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function normalizeBand(raw: unknown, fallback: EqBandParam): EqBandParam {
  if (!raw || typeof raw !== "object") return { ...fallback };
  const band = raw as Partial<EqBandParam>;
  return {
    frequency: clamp(
      typeof band.frequency === "number" ? band.frequency : fallback.frequency,
      EQ_FREQ_MIN,
      EQ_FREQ_MAX,
    ),
    gain: clamp(
      typeof band.gain === "number" ? band.gain : fallback.gain,
      EQ_GAIN_MIN,
      EQ_GAIN_MAX,
    ),
    q: clamp(
      typeof band.q === "number" ? band.q : fallback.q,
      EQ_Q_MIN,
      EQ_Q_MAX,
    ),
  };
}

export function normalizeEqSettings(raw: unknown): EqSettings {
  if (!raw || typeof raw !== "object") return defaultEqSettings();
  const obj = raw as Record<string, unknown>;
  const enabled = Boolean(obj.enabled);
  const presetId = typeof obj.presetId === "string" ? obj.presetId : "flat";
  const defaults = defaultBandParams();

  if (Array.isArray(obj.bands) && obj.bands.length === EQ_BAND_COUNT) {
    return {
      presetId,
      enabled,
      bands: obj.bands.map((band, i) => normalizeBand(band, defaults[i])),
    };
  }

  if (Array.isArray(obj.gains) && obj.gains.length === EQ_BAND_COUNT) {
    const gains = obj.gains as number[];
    return {
      presetId,
      enabled,
      bands: defaults.map((band, i) => ({
        ...band,
        gain: clamp(Number(gains[i]) || 0, EQ_GAIN_MIN, EQ_GAIN_MAX),
      })),
    };
  }

  return defaultEqSettings();
}

export function loadEqFromLocalStorage(): EqSettings {
  if (typeof localStorage === "undefined") return defaultEqSettings();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultEqSettings();
    return normalizeEqSettings(JSON.parse(raw));
  } catch {
    return defaultEqSettings();
  }
}

export function saveEqToLocalStorage(settings: EqSettings) {
  if (typeof localStorage === "undefined") return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
}

export function presetBands(presetId: string): EqBandParam[] {
  const preset = EQ_PRESETS.find((p) => p.id === presetId);
  if (!preset) return defaultBandParams();
  return preset.bands.map((band) => ({ ...band }));
}

export type { EqSettings, EqBandParam, EqPreset } from "./eq-data";
export {
  defaultBandParams,
  defaultEqSettings,
  DEFAULT_BAND_FREQUENCIES,
  EQ_BAND_COUNT,
  EQ_DEFAULT_Q,
  EQ_FREQ_MAX,
  EQ_FREQ_MIN,
  EQ_GAIN_MAX,
  EQ_GAIN_MIN,
  EQ_GAIN_STEP,
  EQ_PRESETS,
  EQ_Q_MAX,
  EQ_Q_MIN,
  EQ_Q_STEP,
  formatFrequency,
  formatGain,
  formatQ,
  freqToNorm,
  normToFreq,
} from "./eq-data";
