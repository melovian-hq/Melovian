// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface Rgb {
  r: number;
  g: number;
  b: number;
}

export const LIGHT_THEME_COLORS = {
  text: "#18181b",
  textMuted: "#52525b",
  textSubtle: "#71717a",
  surface: "#ffffff",
  bg: "#f4f4f5",
} as const;

export const DARK_THEME_COLORS = {
  text: "#f5f5f5",
  textMuted: "#b0b0b0",
  textSubtle: "#8a8a8a",
  surface: "#141414",
  bg: "#090909",
} as const;

export function parseHexColor(color: string): Rgb | null {
  const normalized = color.trim().toLowerCase();
  const match = /^#([0-9a-f]{6})$/.exec(normalized);
  if (!match) return null;
  const value = match[1];
  return {
    r: Number.parseInt(value.slice(0, 2), 16),
    g: Number.parseInt(value.slice(2, 4), 16),
    b: Number.parseInt(value.slice(4, 6), 16),
  };
}

export function relativeLuminance({ r, g, b }: Rgb): number {
  const channel = (value: number) => {
    const unit = value / 255;
    return unit <= 0.03928 ? unit / 12.92 : ((unit + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

export function contrastRatio(foreground: Rgb, background: Rgb): number {
  const fg = relativeLuminance(foreground);
  const bg = relativeLuminance(background);
  const lighter = Math.max(fg, bg);
  const darker = Math.min(fg, bg);
  return (lighter + 0.05) / (darker + 0.05);
}

export function meetsWcagAaNormalText(
  foreground: Rgb,
  background: Rgb,
  minimum = 4.5,
): boolean {
  return contrastRatio(foreground, background) >= minimum;
}

export function applyThemeDataset(theme: "light" | "dark") {
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme;
}

export function parseRgbColor(color: string): Rgb | null {
  const normalized = color.trim().toLowerCase();
  const hex = parseHexColor(normalized);
  if (hex) return hex;

  const rgbMatch = /^rgba?\(([^)]+)\)$/.exec(normalized);
  if (!rgbMatch) return null;

  const parts = rgbMatch[1]
    .split(",")
    .map((part) => Number.parseFloat(part.trim()));
  if (parts.length < 3 || parts.some((part) => Number.isNaN(part))) return null;

  return {
    r: parts[0],
    g: parts[1],
    b: parts[2],
  };
}
