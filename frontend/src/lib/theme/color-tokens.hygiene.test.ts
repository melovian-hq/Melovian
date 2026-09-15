// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { existsSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";
import { listSvelteFiles } from "../../test-fixtures/svelte-files";

// Scans every .svelte file for hard-coded colors. Components must use the
// semantic --jb-* tokens declared in lib/theme/tokens.css so the dark, light,
// and system themes plus accent presets stay consistent. Existing violations
// are pinned in ALLOWLIST so the suite passes today but fails the moment a
// new file, or a new violation in a clean file, appears.

const SRC = join(import.meta.dirname, "../..");

/**
 * Pre-existing violations found by the baseline scan. Each entry names the
 * file (relative to src/) and why it trips the scanner. Remove an entry when
 * the file is migrated to --jb-* tokens. A missing file fails the suite so
 * the list cannot rot.
 */
const ALLOWLIST = new Map<string, string>([
  [
    "lib/components/desktop/WindowControls.svelte",
    "Windows-style close hover #e81123 and white glyph",
  ],
  ["lib/components/layout/BottomNav.svelte", "box-shadow rgb(0 0 0 / 0.12)"],
  [
    "lib/components/music/CollectionHero.svelte",
    "hard-coded hero gradient palette (#4c1d95, #0284c7, ...)",
  ],
  [
    "lib/components/music/CoverArtPlayOverlay.svelte",
    "overlay glyph #fff and color-mix #000",
  ],
  ["lib/components/music/GenreArt.svelte", "rgb() gradient stops"],
  [
    "lib/components/music/HomeShortcutGrid.svelte",
    "#0a0a0a surface and rgb() scrim",
  ],
  [
    "lib/components/music/MusicPlayerFull.svelte",
    "#000 backdrop, rgb() scrims, white text",
  ],
  [
    "lib/components/music/NowPlayingTvMode.svelte",
    "intentionally dark TV chrome (#050505, white)",
  ],
  [
    "lib/components/music/ProgressSeek.svelte",
    "#ffffff handle and rgb() track",
  ],
  [
    "lib/components/settings/ThemeCustomization.svelte",
    "hue slider spectrum hsl() gradient and placeholder hex sample",
  ],
  ["lib/components/ui/Toggle.svelte", "background: white knob"],
  ["pages/MixPage.svelte", "black art-darkening scrim gradient"],
]);

// CSS named colors (CSS Color 4 keyword list). transparent, currentColor,
// and inherit are handled separately and never flagged.
const NAMED_COLORS = new Set(
  (
    "aliceblue antiquewhite aqua aquamarine azure beige bisque black " +
    "blanchedalmond blue blueviolet brown burlywood cadetblue chartreuse " +
    "chocolate coral cornflowerblue cornsilk crimson cyan darkblue darkcyan " +
    "darkgoldenrod darkgray darkgreen darkgrey darkkhaki darkmagenta " +
    "darkolivegreen darkorange darkorchid darkred darksalmon darkseagreen " +
    "darkslateblue darkslategray darkslategrey darkturquoise darkviolet " +
    "deeppink deepskyblue dimgray dimgrey dodgerblue firebrick floralwhite " +
    "forestgreen fuchsia gainsboro ghostwhite gold goldenrod gray green " +
    "greenyellow grey honeydew hotpink indianred indigo ivory khaki lavender " +
    "lavenderblush lawngreen lemonchiffon lightblue lightcoral lightcyan " +
    "lightgoldenrodyellow lightgray lightgreen lightgrey lightpink " +
    "lightsalmon lightseagreen lightskyblue lightslategray lightslategrey " +
    "lightsteelblue lightyellow lime limegreen linen magenta maroon " +
    "mediumaquamarine mediumblue mediumorchid mediumpurple mediumseagreen " +
    "mediumslateblue mediumspringgreen mediumturquoise mediumvioletred " +
    "midnightblue mintcream mistyrose moccasin navajowhite navy oldlace " +
    "olive olivedrab orange orangered orchid palegoldenrod palegreen " +
    "paleturquoise palevioletred papayawhip peachpuff peru pink plum " +
    "powderblue purple rebeccapurple red rosybrown royalblue saddlebrown " +
    "salmon sandybrown seagreen seashell sienna silver skyblue slateblue " +
    "slategray slategrey snow springgreen steelblue tan teal thistle " +
    "tomato turquoise violet wheat white whitesmoke yellow yellowgreen"
  ).split(" "),
);

const SAFE_ATTR_VALUES = new Set([
  "none",
  "currentcolor",
  "transparent",
  "inherit",
  "context-fill",
  "context-stroke",
]);

// Hex colors of length 3, 4, 6, or 8. The lookahead keeps #fade-style
// anchors and longer identifiers from matching. Regexes used with .test()
// stay non-global so lastIndex state cannot skip matches across files.
const hexRE =
  /#(?:[0-9a-fA-F]{8}|[0-9a-fA-F]{6}|[0-9a-fA-F]{3,4})(?![0-9a-zA-Z])/;
// Color functions. color-mix(), light-dark(), and var() are not listed on
// purpose. Literals nested inside them are still caught by hexRE.
const colorFnRE = /\b(?:rgba?|hsla?|oklch|oklab|lab|lch|hwb|color)\s*\(/i;
const styleBlockRE = /<style[^>]*>([\s\S]*?)<\/style>/gi;
const styleAttrRE = /\bstyle\s*=\s*"([^"]*)"/g;
const styleAttrSingleRE = /\bstyle\s*=\s*'([^']*)'/g;
const styleDirectiveRE = /\bstyle:[a-zA-Z-]+\s*=\s*\{([^}]*)\}/g;
const paintAttrRE =
  /\b(?:fill|stroke|stop-color|flood-color|lighting-color)\s*=\s*"([^"]*)"/g;

const namedAlt = [...NAMED_COLORS].join("|");
const namedValueRE = new RegExp(`(?:^|[\\s:,(])(?:${namedAlt})(?![\\w-])`, "i");
const namedQuotedRE = new RegExp(`["'](?:${namedAlt})["']`, "i");

function stripComments(source: string): string {
  return source
    .replace(/<!--[\s\S]*?-->/g, " ")
    .replace(/\/\*[\s\S]*?\*\//g, " ")
    .replace(/\/\/[^\n"]*/g, " ");
}

function fileHasHardCodedColor(file: string): boolean {
  const source = stripComments(readFileSync(file, "utf8"));

  if (hexRE.test(source) || colorFnRE.test(source)) return true;

  const contexts: string[] = [];
  for (const match of source.matchAll(styleBlockRE)) contexts.push(match[1]);
  for (const match of source.matchAll(styleAttrRE)) contexts.push(match[1]);
  for (const match of source.matchAll(styleAttrSingleRE))
    contexts.push(match[1]);

  for (const context of contexts) {
    // Drop var() and color-mix() bodies before the named-color pass so a
    // token reference like var(--jb-danger) or color-mix(in srgb,
    // var(--jb-accent) 65%, transparent) never flags.
    const cleaned = context
      .replace(/var\([^)]*\)/g, " ")
      .replace(/color-mix\([^)]*\)/g, " ");
    if (namedValueRE.test(cleaned)) return true;
  }

  // style:property={...} directives hold JS expressions, so look for quoted
  // color keywords inside them.
  for (const match of source.matchAll(styleDirectiveRE)) {
    if (namedQuotedRE.test(match[1])) return true;
  }

  // SVG paint attributes written as named colors (fill="white"). Hex values
  // already matched hexRE above. currentColor and none stay legal.
  for (const match of source.matchAll(paintAttrRE)) {
    const value = match[1].trim().toLowerCase();
    if (NAMED_COLORS.has(value) && !SAFE_ATTR_VALUES.has(value)) return true;
  }

  return false;
}

function collectOffenders(): string[] {
  return listSvelteFiles(SRC)
    .filter(fileHasHardCodedColor)
    .map((file) => relative(SRC, file))
    .sort();
}

describe("semantic color tokens", () => {
  it("flags no hard-coded colors outside the allowlist baseline", () => {
    const offenders = collectOffenders();
    expect(offenders.length).toBeGreaterThan(0);

    const fresh = offenders.filter((file) => !ALLOWLIST.has(file));
    expect(
      fresh,
      `new hard-coded colors found; use --jb-* tokens from lib/theme/tokens.css instead:\n${fresh.join("\n")}`,
    ).toEqual([]);

    const fixed = [...ALLOWLIST.keys()].filter(
      (file) => existsSync(join(SRC, file)) && !offenders.includes(file),
    );
    if (fixed.length > 0) {
      console.info(
        `allowlist entries now clean, remove them from ALLOWLIST:\n${fixed.join("\n")}`,
      );
    }
  });

  it("keeps every allowlisted file present in the tree", () => {
    const missing = [...ALLOWLIST.keys()].filter(
      (file) => !existsSync(join(SRC, file)),
    );
    expect(
      missing,
      `allowlisted files no longer exist, prune ALLOWLIST:\n${missing.join("\n")}`,
    ).toEqual([]);
  });
});
