// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { ResolvedTheme } from "./accent";

/**
 * Surface palettes. A palette re-tints the private --jb-raw-neutral ramp
 * inside tokens.css while the semantic layer stays untouched, so every
 * component follows without per-palette markup. Every palette ships both a
 * dark and a light ramp, so one pick keeps both appearances consistent.
 */
export interface ThemePalette {
  id: string;
  label: string;
  /** Preview colors for the settings swatch, keyed by resolved theme. */
  dark: { bg: string; surface: string };
  light: { bg: string; surface: string };
}

export const DEFAULT_PALETTE_ID = "default";
export const CUSTOM_PALETTE_ID = "custom";

export const THEME_PALETTES: readonly ThemePalette[] = [
  {
    id: "midnight",
    label: "Midnight",
    dark: { bg: "#060a14", surface: "#0e1628" },
    light: { bg: "#eef1f8", surface: "#ffffff" },
  },
  {
    id: "oled",
    label: "OLED",
    dark: { bg: "#000000", surface: "#121212" },
    light: { bg: "#ffffff", surface: "#f7f7f8" },
  },
  {
    id: "espresso",
    label: "Espresso",
    dark: { bg: "#0f0a07", surface: "#1f150d" },
    light: { bg: "#f7f2ec", surface: "#fffdfb" },
  },
  {
    id: "paper",
    label: "Paper",
    dark: { bg: "#141210", surface: "#1f1c18" },
    light: { bg: "#f4eddd", surface: "#fffdf7" },
  },
  {
    id: "forest",
    label: "Forest",
    dark: { bg: "#070c08", surface: "#101a12" },
    light: { bg: "#eef4ec", surface: "#fdfffd" },
  },
  {
    id: "ocean",
    label: "Ocean",
    dark: { bg: "#060b0f", surface: "#0e1a20" },
    light: { bg: "#ecf3f7", surface: "#fcfeff" },
  },
  {
    id: "rose",
    label: "Rose",
    dark: { bg: "#0f070a", surface: "#1f1016" },
    light: { bg: "#f8eef1", surface: "#fffdfd" },
  },
] as const;

export function palettesForMode(_mode: ResolvedTheme): ThemePalette[] {
  return [...THEME_PALETTES];
}

export function isPaletteIdForMode(id: string, _mode: ResolvedTheme): boolean {
  return (
    id === DEFAULT_PALETTE_ID ||
    id === CUSTOM_PALETTE_ID ||
    THEME_PALETTES.some((palette) => palette.id === id)
  );
}

/**
 * User-authored palette. bg and surface anchor the neutral ramp, accent is
 * optional and falls back to the theme accent. Colors are #rrggbb hex.
 */
export interface CustomPaletteColors {
  bg: string;
  surface: string;
  accent: string;
}

export const DEFAULT_CUSTOM_PALETTE: Record<
  ResolvedTheme,
  CustomPaletteColors
> = {
  dark: { bg: "#0a0a0f", surface: "#14141b", accent: "" },
  light: { bg: "#f4f4f5", surface: "#ffffff", accent: "" },
};

/** Corner radius styles applied as inline --jb-radius-* overrides. */
export type RadiusStyleId = "default" | "sharp" | "square" | "round";

export const DEFAULT_RADIUS_ID = "default" as const;

export const RADIUS_STYLES: readonly {
  id: RadiusStyleId;
  label: string;
}[] = [
  { id: "default", label: "Default" },
  { id: "sharp", label: "Sharp" },
  { id: "square", label: "Square" },
  { id: "round", label: "Round" },
] as const;

export const RADIUS_OVERRIDES: Record<
  Exclude<RadiusStyleId, "default">,
  Record<"sm" | "md" | "lg" | "xl" | "full", string>
> = {
  sharp: {
    sm: "0.1875rem",
    md: "0.25rem",
    lg: "0.375rem",
    xl: "0.5rem",
    full: "9999px",
  },
  square: {
    sm: "0",
    md: "0",
    lg: "0",
    xl: "0",
    full: "0",
  },
  round: {
    sm: "0.5625rem",
    md: "0.75rem",
    lg: "1.125rem",
    xl: "1.5rem",
    full: "9999px",
  },
};

/** Interface size applied as a root font-size percentage. */
export type UiSizeId = "default" | "compact" | "large";

export const DEFAULT_UI_SIZE_ID = "default" as const;

export const UI_SIZES: readonly { id: UiSizeId; label: string }[] = [
  { id: "compact", label: "Compact" },
  { id: "default", label: "Default" },
  { id: "large", label: "Large" },
] as const;

export const UI_SIZE_FONT_SCALE: Record<
  Exclude<UiSizeId, "default">,
  string
> = {
  compact: "90%",
  large: "112.5%",
};
