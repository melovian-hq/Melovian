// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  EQ_FREQ_MAX,
  EQ_FREQ_MIN,
  EQ_GAIN_MAX,
  EQ_GAIN_MIN,
  type EqBandParam,
} from "./eq-data";

const SAMPLE_RATE = 44100;

function peakingCoefficients(f0: number, Q: number, gainDb: number) {
  const A = Math.pow(10, gainDb / 40);
  const w0 = (2 * Math.PI * f0) / SAMPLE_RATE;
  const cosW0 = Math.cos(w0);
  const alpha = Math.sin(w0) / (2 * Q);

  const b0 = 1 + alpha * A;
  const b1 = -2 * cosW0;
  const b2 = 1 - alpha * A;
  const a0 = 1 + alpha / A;
  const a1 = -2 * cosW0;
  const a2 = 1 - alpha / A;

  return { b0, b1, b2, a0, a1, a2 };
}

function biquadMagnitude(
  b0: number,
  b1: number,
  b2: number,
  a0: number,
  a1: number,
  a2: number,
  w: number,
): number {
  const cosW = Math.cos(w);
  const sinW = Math.sin(w);
  const cos2W = Math.cos(2 * w);
  const sin2W = Math.sin(2 * w);

  const nr = b0 + b1 * cosW + b2 * cos2W;
  const ni = -(b1 * sinW + b2 * sin2W);
  const dr = a0 + a1 * cosW + a2 * cos2W;
  const di = -(a1 * sinW + a2 * sin2W);

  return Math.sqrt((nr * nr + ni * ni) / (dr * dr + di * di));
}

function bandMagnitude(band: EqBandParam, freq: number): number {
  const { b0, b1, b2, a0, a1, a2 } = peakingCoefficients(
    band.frequency,
    band.q,
    band.gain,
  );
  const w = (2 * Math.PI * freq) / SAMPLE_RATE;
  return biquadMagnitude(b0, b1, b2, a0, a1, a2, w);
}

export interface EqCurvePoint {
  frequency: number;
  gainDb: number;
}

function combinedLinearGain(bands: EqBandParam[], frequency: number): number {
  let linear = 1;
  for (const band of bands) {
    linear *= bandMagnitude(band, frequency);
  }
  return linear;
}

export function combinedGainDbAt(
  bands: EqBandParam[],
  frequency: number,
): number {
  const linear = combinedLinearGain(bands, frequency);
  if (linear <= 0) return EQ_GAIN_MIN;
  return 20 * Math.log10(linear);
}

export function computeCombinedCurve(
  bands: EqBandParam[],
  points = 128,
): EqCurvePoint[] {
  const minLog = Math.log10(EQ_FREQ_MIN);
  const maxLog = Math.log10(EQ_FREQ_MAX);
  const result: EqCurvePoint[] = [];

  for (let i = 0; i <= points; i++) {
    const t = i / points;
    const frequency = Math.pow(10, minLog + t * (maxLog - minLog));
    const gainDb = combinedGainDbAt(bands, frequency);
    result.push({ frequency, gainDb });
  }

  return result;
}

export function freqToX(frequency: number, width: number): number {
  const minLog = Math.log10(EQ_FREQ_MIN);
  const maxLog = Math.log10(EQ_FREQ_MAX);
  const clamped = Math.min(EQ_FREQ_MAX, Math.max(EQ_FREQ_MIN, frequency));
  return ((Math.log10(clamped) - minLog) / (maxLog - minLog)) * width;
}

export function xToFreq(x: number, width: number): number {
  const minLog = Math.log10(EQ_FREQ_MIN);
  const maxLog = Math.log10(EQ_FREQ_MAX);
  const t = Math.min(width, Math.max(0, x)) / width;
  return Math.pow(10, minLog + t * (maxLog - minLog));
}

export function gainToY(gainDb: number, height: number): number {
  const clamped = Math.min(EQ_GAIN_MAX, Math.max(EQ_GAIN_MIN, gainDb));
  const t = (EQ_GAIN_MAX - clamped) / (EQ_GAIN_MAX - EQ_GAIN_MIN);
  return t * height;
}

export function yToGain(y: number, height: number): number {
  const t = Math.min(height, Math.max(0, y)) / height;
  const gainDb = EQ_GAIN_MAX - t * (EQ_GAIN_MAX - EQ_GAIN_MIN);
  return Math.round(gainDb * 10) / 10;
}

export function curvePath(
  points: EqCurvePoint[],
  width: number,
  height: number,
): string {
  if (points.length === 0) return "";
  const segments = points.map((point, i) => {
    const x = freqToX(point.frequency, width);
    const y = gainToY(point.gainDb, height);
    return `${i === 0 ? "M" : "L"}${x.toFixed(2)},${y.toFixed(2)}`;
  });
  return segments.join(" ");
}

export function curveAreaPath(
  points: EqCurvePoint[],
  width: number,
  height: number,
): string {
  if (points.length === 0) return "";
  const line = curvePath(points, width, height);
  const zeroY = gainToY(0, height);
  const lastX = freqToX(points[points.length - 1]!.frequency, width);
  const firstX = freqToX(points[0]!.frequency, width);
  return `${line} L${lastX.toFixed(2)},${zeroY.toFixed(2)} L${firstX.toFixed(2)},${zeroY.toFixed(2)} Z`;
}
