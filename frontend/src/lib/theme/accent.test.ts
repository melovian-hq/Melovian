// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  ACCENT_PRESETS,
  accentTextFor,
  CUSTOM_ACCENT_ID,
  DEFAULT_ACCENT_HUE,
  DEFAULT_ACCENT_PRESET_ID,
  getAccentPreset,
  hexToHue,
  hueToAccentHex,
  isAccentPresetId,
  normalizeHue,
  presetAccentHex,
} from "./accent";
import {
  contrastRatio,
  DARK_THEME_COLORS,
  LIGHT_THEME_COLORS,
  parseHexColor,
} from "./contrast";

const surfaces = {
  dark: DARK_THEME_COLORS.surface,
  light: LIGHT_THEME_COLORS.surface,
} as const;

function ratio(a: string, b: string): number {
  const ra = parseHexColor(a);
  const rb = parseHexColor(b);
  if (!ra || !rb) throw new Error(`unparseable color pair ${a} ${b}`);
  return contrastRatio(ra, rb);
}

describe("accent presets", () => {
  it("exposes a violet default preset in the 262-272 hue family", () => {
    const aurora = getAccentPreset(DEFAULT_ACCENT_PRESET_ID);
    expect(aurora).toBeDefined();
    expect(hexToHue(aurora!.dark)).toBeGreaterThanOrEqual(262);
    expect(hexToHue(aurora!.dark)).toBeLessThanOrEqual(272);
  });

  it("resolves one tuned hex per theme and falls back to the default", () => {
    for (const preset of ACCENT_PRESETS) {
      expect(presetAccentHex(preset.id, "dark")).toBe(preset.dark);
      expect(presetAccentHex(preset.id, "light")).toBe(preset.light);
    }
    expect(presetAccentHex("bogus", "dark")).toBe(
      presetAccentHex(DEFAULT_ACCENT_PRESET_ID, "dark"),
    );
  });

  it("validates preset ids including the custom sentinel", () => {
    for (const preset of ACCENT_PRESETS) {
      expect(isAccentPresetId(preset.id)).toBe(true);
    }
    expect(isAccentPresetId(CUSTOM_ACCENT_ID)).toBe(true);
    expect(isAccentPresetId("bogus")).toBe(false);
  });

  it("keeps every preset readable as accent text on its theme surface", () => {
    for (const preset of ACCENT_PRESETS) {
      for (const resolved of ["dark", "light"] as const) {
        const accent = presetAccentHex(preset.id, resolved);
        expect(
          ratio(accent, surfaces[resolved]),
          `${preset.id} ${resolved} on surface`,
        ).toBeGreaterThanOrEqual(4.5);
        expect(
          ratio(accentTextFor(accent), accent),
          `${preset.id} ${resolved} accent text`,
        ).toBeGreaterThanOrEqual(4.5);
      }
    }
  });
});

describe("custom hue derivation", () => {
  const hues = [0, 30, 60, 120, 180, 210, 240, 266, 300, 330];

  it("derives contrast-safe accents for sampled hues in both themes", () => {
    for (const hue of hues) {
      for (const resolved of ["dark", "light"] as const) {
        const accent = hueToAccentHex(hue, resolved);
        expect(
          ratio(accent, surfaces[resolved]),
          `hue ${hue} ${resolved} on surface`,
        ).toBeGreaterThanOrEqual(4.5);
        expect(
          ratio(accentTextFor(accent), accent),
          `hue ${hue} ${resolved} accent text`,
        ).toBeGreaterThanOrEqual(4.5);
      }
    }
  });

  it("derives the default hue close to the aurora preset", () => {
    const derived = hueToAccentHex(DEFAULT_ACCENT_HUE, "dark");
    expect(hexToHue(derived)).toBe(DEFAULT_ACCENT_HUE);
  });

  it("normalizes out-of-range hues", () => {
    expect(normalizeHue(-10)).toBe(350);
    expect(normalizeHue(370)).toBe(10);
    expect(normalizeHue(Number.NaN)).toBe(DEFAULT_ACCENT_HUE);
  });
});

describe("hexToHue", () => {
  it("extracts hues from hex colors", () => {
    expect(hexToHue("#ff0000")).toBe(0);
    expect(hexToHue("#00ff00")).toBe(120);
    expect(hexToHue("#0000ff")).toBe(240);
    expect(hexToHue("#a468f3")).toBe(266);
  });

  it("falls back to the default hue for invalid or gray input", () => {
    expect(hexToHue("nope")).toBe(DEFAULT_ACCENT_HUE);
    expect(hexToHue("#888888")).toBe(DEFAULT_ACCENT_HUE);
  });
});
