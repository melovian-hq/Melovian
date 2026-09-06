// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface EqBandParam {
  frequency: number;
  gain: number;
  q: number;
}

export interface EqPreset {
  id: string;
  name: string;
  bands: EqBandParam[];
}

export interface EqSettings {
  presetId: string;
  bands: EqBandParam[];
  enabled: boolean;
}

export const EQ_BAND_COUNT = 10;

export const EQ_GAIN_MIN = -24;
export const EQ_GAIN_MAX = 24;
export const EQ_GAIN_STEP = 0.1;

export const EQ_FREQ_MIN = 20;
export const EQ_FREQ_MAX = 20000;

export const EQ_Q_MIN = 0.25;
export const EQ_Q_MAX = 12;
export const EQ_Q_STEP = 0.05;
export const EQ_DEFAULT_Q = 1;

export const DEFAULT_BAND_FREQUENCIES = [
  60, 170, 310, 600, 1000, 3000, 6000, 12000, 14000, 16000,
] as const;

export function formatFrequency(hz: number): string {
  if (hz >= 1000) {
    const k = hz / 1000;
    return k >= 10 ? `${Math.round(k)}k` : `${k.toFixed(k >= 1 ? 1 : 2)}k`;
  }
  return `${Math.round(hz)}`;
}

export function formatGain(db: number): string {
  const rounded = Math.round(db * 10) / 10;
  return `${rounded > 0 ? "+" : ""}${rounded.toFixed(1)}`;
}

export function formatQ(q: number): string {
  return (Math.round(q * 100) / 100).toFixed(2);
}

export function freqToNorm(hz: number): number {
  const minLog = Math.log10(EQ_FREQ_MIN);
  const maxLog = Math.log10(EQ_FREQ_MAX);
  const clamped = Math.min(EQ_FREQ_MAX, Math.max(EQ_FREQ_MIN, hz));
  return ((Math.log10(clamped) - minLog) / (maxLog - minLog)) * 1000;
}

export function normToFreq(norm: number): number {
  const minLog = Math.log10(EQ_FREQ_MIN);
  const maxLog = Math.log10(EQ_FREQ_MAX);
  const t = Math.min(1000, Math.max(0, norm)) / 1000;
  return Math.round(Math.pow(10, minLog + t * (maxLog - minLog)));
}

export function defaultBandParams(gain = 0, q = EQ_DEFAULT_Q): EqBandParam[] {
  return DEFAULT_BAND_FREQUENCIES.map((frequency) => ({
    frequency,
    gain,
    q,
  }));
}

function presetFromGains(id: string, name: string, gains: number[]): EqPreset {
  return {
    id,
    name,
    bands: DEFAULT_BAND_FREQUENCIES.map((frequency, i) => ({
      frequency,
      gain: gains[i] ?? 0,
      q: EQ_DEFAULT_Q,
    })),
  };
}

export const EQ_PRESETS: EqPreset[] = [
  presetFromGains("flat", "Flat", [0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
  presetFromGains("bass", "Bass Boost", [6, 5, 3, 1, 0, 0, 0, 0, 0, 0]),
  presetFromGains("treble", "Treble", [0, 0, 0, 0, 1, 2, 4, 5, 4, 3]),
  presetFromGains("vocal", "Vocal", [-2, -1, 0, 2, 4, 4, 3, 1, 0, -1]),
  presetFromGains("rock", "Rock", [4, 3, 1, 0, -1, 0, 2, 3, 3, 2]),
  presetFromGains("electronic", "Electronic", [5, 4, 1, 0, -2, 2, 1, 3, 4, 5]),
];

export function defaultEqSettings(): EqSettings {
  const flat = EQ_PRESETS[0];
  return {
    presetId: flat.id,
    bands: flat.bands.map((band) => ({ ...band })),
    enabled: false,
  };
}
