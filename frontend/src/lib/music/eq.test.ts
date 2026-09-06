// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  defaultBandParams,
  defaultEqSettings,
  EQ_BAND_COUNT,
  normalizeEqSettings,
  presetBands,
} from "./eq";

describe("normalizeEqSettings", () => {
  it("returns defaults for invalid input", () => {
    expect(normalizeEqSettings(null)).toEqual(defaultEqSettings());
    expect(normalizeEqSettings("bad")).toEqual(defaultEqSettings());
  });

  it("normalizes full band payloads", () => {
    const defaults = defaultBandParams();
    const bands = defaults.map((band, index) => ({
      ...band,
      gain: index === 0 ? 99 : band.gain,
      frequency: index === 1 ? 10 : band.frequency,
    }));

    const normalized = normalizeEqSettings({
      enabled: true,
      presetId: "custom",
      bands,
    });

    expect(normalized.enabled).toBe(true);
    expect(normalized.presetId).toBe("custom");
    expect(normalized.bands).toHaveLength(EQ_BAND_COUNT);
    expect(normalized.bands[0]?.gain).toBe(24);
    expect(normalized.bands[1]?.frequency).toBe(20);
  });

  it("migrates legacy gains arrays", () => {
    const normalized = normalizeEqSettings({
      enabled: false,
      presetId: "flat",
      gains: Array(EQ_BAND_COUNT).fill(6),
    });

    expect(normalized.bands.every((band) => band.gain === 6)).toBe(true);
  });

  it("falls back when band count is wrong", () => {
    expect(
      normalizeEqSettings({
        enabled: true,
        presetId: "flat",
        bands: [{ frequency: 100, gain: 1, q: 1 }],
      }),
    ).toEqual(defaultEqSettings());
  });
});

describe("presetBands", () => {
  it("returns preset bands or defaults for unknown presets", () => {
    const rock = presetBands("rock");
    expect(rock).toHaveLength(EQ_BAND_COUNT);
    expect(presetBands("missing-preset")).toEqual(defaultBandParams());
  });
});
