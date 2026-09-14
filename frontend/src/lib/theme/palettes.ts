// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { ResolvedTheme } from "./accent";

/**
 * Surface palettes. A palette re-tints the private --jb-raw-neutral ramp
 * inside tokens.css while the semantic layer stays untouched, so every
 * component follows without per-palette markup. Palettes are keyed to a
 * resolved theme. Picking a dark palette only applies while the resolved
 * theme is dark.
 */
export interface ThemePalette {
  id: string;
  label: string;
  mode: ResolvedTheme;
  /** Preview colors for the settings swatch. */
  bg: string;
  surface: string;
}

export const DEFAULT_PALETTE_ID = "default";

export const THEME_PALETTES: readonly ThemePalette[] = [
  {
    id: "midnight",
    label: "Midnight",
    mode: "dark",
    bg: "#060a14",
    surface: "#0e1628",
  },
  {
    id: "oled",
    label: "OLED",
    mode: "dark",
    bg: "#000000",
    surface: "#121212",
  },
  {
    id: "espresso",
    label: "Espresso",
    mode: "dark",
    bg: "#0f0a07",
    surface: "#1f150d",
  },
  {
    id: "paper",
    label: "Paper",
    mode: "light",
    bg: "#f4eddd",
    surface: "#fffdf7",
  },
] as const;

export function palettesForMode(mode: ResolvedTheme): ThemePalette[] {
  return THEME_PALETTES.filter((palette) => palette.mode === mode);
}

export function isPaletteIdForMode(id: string, mode: ResolvedTheme): boolean {
  return (
    id === DEFAULT_PALETTE_ID ||
    THEME_PALETTES.some((palette) => palette.id === id && palette.mode === mode)
  );
}

/** Corner radius styles applied as inline --jb-radius-* overrides. */
export type RadiusStyleId = "default" | "sharp" | "round";

export const DEFAULT_RADIUS_ID = "default" as const;

export const RADIUS_STYLES: readonly {
  id: RadiusStyleId;
  label: string;
}[] = [
  { id: "default", label: "Default" },
  { id: "sharp", label: "Sharp" },
  { id: "round", label: "Round" },
] as const;

export const RADIUS_OVERRIDES: Record<
  Exclude<RadiusStyleId, "default">,
  Record<"sm" | "md" | "lg" | "xl", string>
> = {
  sharp: {
    sm: "0.1875rem",
    md: "0.25rem",
    lg: "0.375rem",
    xl: "0.5rem",
  },
  round: {
    sm: "0.5625rem",
    md: "0.75rem",
    lg: "1.125rem",
    xl: "1.5rem",
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
