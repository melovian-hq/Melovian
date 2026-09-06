// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  contrastRatio,
  DARK_THEME_COLORS,
  LIGHT_THEME_COLORS,
  meetsWcagAaNormalText,
  parseHexColor,
} from "./contrast";

const themeDir = path.dirname(fileURLToPath(import.meta.url));
const componentsDir = path.resolve(themeDir, "../components/music");

function readComponent(name: string) {
  return readFileSync(path.join(componentsDir, name), "utf8");
}

function readTokensCss() {
  return readFileSync(path.join(themeDir, "tokens.css"), "utf8");
}

function extractThemeBlock(css: string, theme: "light" | "dark") {
  const match = new RegExp(
    `\\[data-theme="${theme}"\\]\\s*\\{([\\s\\S]*?)\\n\\s*\\}`,
    "m",
  ).exec(css);
  return match?.[1] ?? "";
}

describe("contrast", () => {
  it("computes known contrast ratios", () => {
    const white = parseHexColor("#ffffff");
    const black = parseHexColor("#000000");
    expect(white && black && contrastRatio(black, white)).toBeCloseTo(21, 0);
  });

  it("keeps primary music text readable on surfaces in light mode", () => {
    const text = parseHexColor(LIGHT_THEME_COLORS.text);
    const surface = parseHexColor(LIGHT_THEME_COLORS.surface);
    const muted = parseHexColor(LIGHT_THEME_COLORS.textMuted);
    const subtle = parseHexColor(LIGHT_THEME_COLORS.textSubtle);
    expect(text && surface && meetsWcagAaNormalText(text, surface)).toBe(true);
    expect(muted && surface && meetsWcagAaNormalText(muted, surface)).toBe(
      true,
    );
    expect(subtle && surface && meetsWcagAaNormalText(subtle, surface)).toBe(
      true,
    );
  });

  it("keeps primary music text readable on surfaces in dark mode", () => {
    const text = parseHexColor(DARK_THEME_COLORS.text);
    const surface = parseHexColor(DARK_THEME_COLORS.surface);
    const muted = parseHexColor(DARK_THEME_COLORS.textMuted);
    const subtle = parseHexColor(DARK_THEME_COLORS.textSubtle);
    const bg = parseHexColor(DARK_THEME_COLORS.bg);
    expect(text && surface && meetsWcagAaNormalText(text, surface)).toBe(true);
    expect(text && bg && meetsWcagAaNormalText(text, bg)).toBe(true);
    expect(muted && surface && meetsWcagAaNormalText(muted, surface)).toBe(
      true,
    );
    expect(subtle && surface && meetsWcagAaNormalText(subtle, surface)).toBe(
      true,
    );
  });

  it("defines semantic colors only on data-theme selectors", () => {
    const css = readTokensCss();
    const rootBlock = /:root\s*\{([^}]*)\}/s.exec(css)?.[1] ?? "";
    expect(rootBlock).not.toMatch(/--jb-text:/);
    expect(rootBlock).not.toMatch(/--jb-bg:/);
    expect(extractThemeBlock(css, "light")).toMatch(/--jb-text:/);
    expect(extractThemeBlock(css, "dark")).toMatch(/--jb-text:/);
  });

  it("keeps token constants aligned with tokens.css", () => {
    const css = readTokensCss();
    for (const [theme, colors] of [
      ["light", LIGHT_THEME_COLORS],
      ["dark", DARK_THEME_COLORS],
    ] as const) {
      const block = extractThemeBlock(css, theme);
      for (const [token, value] of Object.entries(colors)) {
        const cssName =
          token === "text"
            ? "--jb-text"
            : token === "textMuted"
              ? "--jb-text-muted"
              : token === "textSubtle"
                ? "--jb-text-subtle"
                : token === "surface"
                  ? "--jb-surface"
                  : "--jb-bg";
        expect(block).toContain(`${cssName}: ${value}`);
      }
    }
  });

  it("binds song and album metadata to theme text tokens", () => {
    const trackRow = readComponent("TrackRow.svelte");
    const albumCard = readComponent("AlbumCard.svelte");
    const historyRow = readComponent("ListenHistoryRow.svelte");

    expect(trackRow).toMatch(
      /\.track-row__title\s*\{[^}]*color:\s*var\(--jb-text\)/s,
    );
    expect(albumCard).toMatch(
      /\.album-card__title\s*\{[^}]*color:\s*var\(--jb-text\)/s,
    );
    expect(historyRow).toMatch(
      /\.history-row__title\s*\{[^}]*color:\s*var\(--jb-text\)/s,
    );
  });
});
