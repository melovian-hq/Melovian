// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createBoundedMap } from "$lib/core/bounded-cache";

const GENRE_PALETTES: Record<string, [string, string, string]> = {
  rock: ["#1a1a2e", "#e94560", "#533483"],
  pop: ["#ff6b9d", "#c44569", "#f8b500"],
  jazz: ["#2d132c", "#ee4540", "#801336"],
  classical: ["#1b262c", "#0f4c75", "#bbe1fa"],
  electronic: ["#0f0c29", "#302b63", "#24243e"],
  "hip hop": ["#f12711", "#f5af19", "#1a1a1a"],
  "hip-hop": ["#f12711", "#f5af19", "#1a1a1a"],
  rap: ["#f12711", "#f5af19", "#1a1a1a"],
  metal: ["#232526", "#414345", "#8b0000"],
  country: ["#8b4513", "#daa520", "#228b22"],
  blues: ["#141e30", "#243b55", "#4a90d9"],
  folk: ["#5d4037", "#8d6e63", "#a5d6a7"],
  reggae: ["#f7971e", "#ffd200", "#11998e"],
  punk: ["#ff0084", "#330867", "#000000"],
  soul: ["#834d9b", "#d04ed6", "#ffd89b"],
  rnb: ["#834d9b", "#d04ed6", "#ffd89b"],
  "r&b": ["#834d9b", "#d04ed6", "#ffd89b"],
  ambient: ["#0f2027", "#203a43", "#2c5364"],
  soundtrack: ["#373b44", "#4286f4", "#373b44"],
  latin: ["#f7971e", "#ffd200", "#e53935"],
  world: ["#11998e", "#38ef7d", "#f7971e"],
  disco: ["#ff0084", "#330867", "#ffd89b"],
  indie: ["#667eea", "#764ba2", "#f093fb"],
  alternative: ["#434343", "#667eea", "#764ba2"],
};

const PALETTE_PATTERNS = Object.entries(GENRE_PALETTES).sort(
  (a, b) => b[0].length - a[0].length,
);

const ART_URL_CACHE_MAX_ENTRIES = 500;

const artUrlCache = createBoundedMap<string, string>(ART_URL_CACHE_MAX_ENTRIES);

function hashString(value: string): number {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}

function normalizeGenre(name: string): string {
  return name.trim().toLowerCase();
}

export function genrePalette(name: string): [string, string, string] {
  const key = normalizeGenre(name);
  const exact = GENRE_PALETTES[key];
  if (exact) return exact;

  for (const [pattern, palette] of PALETTE_PATTERNS) {
    if (key.includes(pattern)) return palette;
  }

  const hash = hashString(key);
  const hue = hash % 360;
  return [
    `hsl(${hue} 45% 18%)`,
    `hsl(${(hue + 40) % 360} 65% 45%)`,
    `hsl(${(hue + 80) % 360} 55% 32%)`,
  ];
}

function buildArtUrl(name: string, cells = 8): string {
  const key = normalizeGenre(name);
  const cached = artUrlCache.get(key);
  if (cached) return cached;

  const [bg, accent, mid] = genrePalette(name);
  const hash = hashString(key);
  const parts: string[] = [];
  const last = cells - 1;

  for (let y = 0; y < cells; y++) {
    for (let x = 0; x < cells; x++) {
      const bit = (hash >> ((x + y * cells) % 32)) & 1;
      const corner = (x === 0 || x === last) && (y === 0 || y === last);
      const fill = corner ? accent : bit ? mid : bg;
      parts.push(
        `<rect x='${x}' y='${y}' width='1' height='1' fill='${fill}'/>`,
      );
    }
  }

  const svg = `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 ${cells} ${cells}' shape-rendering='crispEdges'>${parts.join("")}</svg>`;
  const url = `data:image/svg+xml,${encodeURIComponent(svg)}`;
  artUrlCache.set(key, url);
  return url;
}

export function genreArtUrl(name: string): string {
  return buildArtUrl(name);
}

export function prewarmGenreArt(names: readonly string[]): void {
  for (const name of names) {
    genreArtUrl(name);
  }
}

export function genreInitial(name: string): string {
  const trimmed = name.trim();
  if (!trimmed) return "?";
  return trimmed.charAt(0).toUpperCase();
}
