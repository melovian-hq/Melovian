// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  contrastRatio,
  DARK_THEME_COLORS,
  LIGHT_THEME_COLORS,
  parseHexColor,
} from "./contrast";

export type ResolvedTheme = "light" | "dark";

/**
 * An accent preset ships one tuned hex per resolved theme. Presets are the
 * default --jb-accent source. Selecting the default preset applies no
 * inline override at all, so tokens.css keeps full control.
 */
export interface AccentPreset {
  id: string;
  label: string;
  dark: string;
  light: string;
}

export const DEFAULT_ACCENT_PRESET_ID = "aurora";
export const CUSTOM_ACCENT_ID = "custom";
export const DEFAULT_ACCENT_HUE = 266;

/**
 * Brand accent presets. Dark values keep at least 4.5:1 contrast against
 * the dark surface so accents stay readable when used as text or icons.
 * Light values keep at least 4.5:1 against white surfaces.
 */
export const ACCENT_PRESETS: readonly AccentPreset[] = [
  { id: "aurora", label: "Aurora", dark: "#a468f3", light: "#6f25d0" },
  { id: "ocean", label: "Ocean", dark: "#2dd4bf", light: "#0f766e" },
  { id: "forest", label: "Forest", dark: "#4ade80", light: "#15803d" },
  { id: "lime", label: "Lime", dark: "#a3e635", light: "#4d7c0f" },
  { id: "amber", label: "Amber", dark: "#fbbf24", light: "#b45309" },
  { id: "sunset", label: "Sunset", dark: "#fb923c", light: "#c2410c" },
  { id: "crimson", label: "Crimson", dark: "#f87171", light: "#b91c1c" },
  { id: "rose", label: "Rose", dark: "#fb7185", light: "#e11d48" },
  { id: "magenta", label: "Magenta", dark: "#e879f9", light: "#a21caf" },
  { id: "violet", label: "Violet", dark: "#c084fc", light: "#7e22ce" },
  { id: "sky", label: "Sky", dark: "#38bdf8", light: "#0369a1" },
  { id: "graphite", label: "Graphite", dark: "#a1a1aa", light: "#52525b" },
] as const;

/** Ink used on bright accents in dark theme, mirrors --jb-raw-accent-text. */
const ACCENT_TEXT_INK = "#171321";
const ACCENT_TEXT_LIGHT = "#ffffff";

const HEX_COLOR_RE = /^#[0-9a-fA-F]{6}$/;

export function isHexColor(value: string): boolean {
  return HEX_COLOR_RE.test(value.trim());
}

export function getAccentPreset(id: string): AccentPreset | undefined {
  return ACCENT_PRESETS.find((preset) => preset.id === id);
}

export function isAccentPresetId(id: string): boolean {
  return id === CUSTOM_ACCENT_ID || getAccentPreset(id) !== undefined;
}

/** Resolved accent hex for a preset and theme. Falls back to the default. */
export function presetAccentHex(id: string, resolved: ResolvedTheme): string {
  const preset =
    getAccentPreset(id) ?? getAccentPreset(DEFAULT_ACCENT_PRESET_ID);
  return preset ? preset[resolved] : "#a468f3";
}

export function normalizeHue(hue: number): number {
  if (!Number.isFinite(hue)) return DEFAULT_ACCENT_HUE;
  return ((Math.round(hue) % 360) + 360) % 360;
}

/** Pick the accent foreground that best contrasts with the accent fill. */
export function accentTextFor(accentHex: string): string {
  const accent = parseHexColor(accentHex);
  if (!accent) return ACCENT_TEXT_LIGHT;
  let best = ACCENT_TEXT_LIGHT;
  let bestRatio = 0;
  for (const candidate of [ACCENT_TEXT_INK, ACCENT_TEXT_LIGHT]) {
    const rgb = parseHexColor(candidate);
    if (!rgb) continue;
    const ratio = contrastRatio(rgb, accent);
    if (ratio > bestRatio) {
      best = candidate;
      bestRatio = ratio;
    }
  }
  return best;
}

function hslToHex(hue: number, saturation: number, lightness: number): string {
  const s = saturation / 100;
  const l = lightness / 100;
  const k = (n: number) => (n + hue / 30) % 12;
  const a = s * Math.min(l, 1 - l);
  const f = (n: number) =>
    l - a * Math.max(-1, Math.min(k(n) - 3, 9 - k(n), 1));
  const toHex = (channel: number) =>
    Math.round(channel * 255)
      .toString(16)
      .padStart(2, "0");
  return `#${toHex(f(0))}${toHex(f(8))}${toHex(f(4))}`;
}

/** Extract the hue component of a hex color, for custom accent seeding. */
export function hexToHue(hex: string): number {
  const rgb = parseHexColor(hex);
  if (!rgb) return DEFAULT_ACCENT_HUE;
  const r = rgb.r / 255;
  const g = rgb.g / 255;
  const b = rgb.b / 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const delta = max - min;
  if (delta === 0) return DEFAULT_ACCENT_HUE;
  let hue: number;
  if (max === r) {
    hue = 60 * (((g - b) / delta) % 6);
  } else if (max === g) {
    hue = 60 * ((b - r) / delta + 2);
  } else {
    hue = 60 * ((r - g) / delta + 4);
  }
  return normalizeHue(hue);
}

/**
 * Derive a contrast-safe accent hex from a hue. Dark accents start
 * saturated and lighten until they clear 4.5:1 on the dark surface and
 * keep the dark ink readable on the accent fill. Light accents start
 * mid-tone and darken until they clear 4.5:1 on the light surface,
 * which also guarantees white text passes on the accent fill.
 */
export function hueToAccentHex(hue: number, resolved: ResolvedTheme): string {
  const h = normalizeHue(hue);
  const surfaceHex =
    resolved === "dark"
      ? DARK_THEME_COLORS.surface
      : LIGHT_THEME_COLORS.surface;
  const surface = parseHexColor(surfaceHex);
  if (!surface) return presetAccentHex(DEFAULT_ACCENT_PRESET_ID, resolved);

  if (resolved === "dark") {
    const ink = parseHexColor(ACCENT_TEXT_INK);
    for (let lightness = 68; lightness <= 92; lightness += 2) {
      const hex = hslToHex(h, 84, lightness);
      const rgb = parseHexColor(hex);
      if (
        rgb &&
        ink &&
        contrastRatio(rgb, surface) >= 4.5 &&
        contrastRatio(ink, rgb) >= 4.5
      ) {
        return hex;
      }
    }
    return hslToHex(h, 84, 92);
  }

  for (let lightness = 46; lightness >= 16; lightness -= 2) {
    const hex = hslToHex(h, 72, lightness);
    const rgb = parseHexColor(hex);
    if (rgb && contrastRatio(rgb, surface) >= 4.5) return hex;
  }
  return hslToHex(h, 72, 16);
}
