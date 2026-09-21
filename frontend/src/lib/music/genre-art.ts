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

/**
 * Genre glyph map, longest substring match wins. Icon names must stay
 * string literals so scripts/gen-icon-data.cjs bundles them.
 */
const GENRE_ICONS: Record<string, string> = {
  "hip hop": "mdi:microphone-variant",
  "hip-hop": "mdi:microphone-variant",
  rap: "mdi:microphone-variant",
  trap: "mdi:microphone-variant",
  reggaeton: "mdi:microphone-variant",
  "k-pop": "mdi:star-outline",
  "j-pop": "mdi:star-outline",
  metal: "mdi:skull-outline",
  hardcore: "mdi:skull-crossbones",
  punk: "mdi:skull-crossbones",
  rock: "mdi:guitar-electric",
  grunge: "mdi:guitar-electric",
  pop: "mdi:microphone",
  karaoke: "mdi:microphone",
  jazz: "mdi:saxophone",
  blues: "mdi:trumpet",
  swing: "mdi:trumpet",
  "big band": "mdi:trumpet",
  classical: "mdi:violin",
  orchestra: "mdi:violin",
  opera: "mdi:drama-masks",
  soundtrack: "mdi:filmstrip",
  score: "mdi:filmstrip",
  film: "mdi:filmstrip",
  electronic: "mdi:sine-wave",
  techno: "mdi:sine-wave",
  house: "mdi:sine-wave",
  trance: "mdi:sine-wave",
  dubstep: "mdi:sine-wave",
  synth: "mdi:sine-wave",
  edm: "mdi:sine-wave",
  ambient: "mdi:waves",
  chill: "mdi:waves",
  "new age": "mdi:moon-waning-crescent",
  meditation: "mdi:moon-waning-crescent",
  country: "mdi:guitar-acoustic",
  folk: "mdi:guitar-pick",
  acoustic: "mdi:guitar-pick",
  indie: "mdi:guitar-pick-outline",
  alternative: "mdi:guitar-pick-outline",
  reggae: "mdi:palm-tree",
  soul: "mdi:heart-outline",
  rnb: "mdi:heart-outline",
  "r&b": "mdi:heart-outline",
  funk: "mdi:record-player",
  disco: "mdi:disc",
  dance: "mdi:disc",
  latin: "mdi:fire",
  salsa: "mdi:fire",
  world: "mdi:earth",
  gospel: "mdi:church-outline",
  christian: "mdi:church-outline",
  worship: "mdi:church-outline",
  podcast: "mdi:podcast",
  spoken: "mdi:podcast",
  audiobook: "mdi:podcast",
  holiday: "mdi:pine-tree",
  christmas: "mdi:pine-tree",
  children: "mdi:balloon",
  kids: "mdi:balloon",
  instrumental: "mdi:piano",
  piano: "mdi:piano",
  radio: "mdi:radio",
  drum: "mdi:speaker",
  bass: "mdi:speaker",
};

const GENRE_ICON_PATTERNS = Object.entries(GENRE_ICONS).sort(
  (a, b) => b[0].length - a[0].length,
);

const DEFAULT_GENRE_ICON = "mdi:music-note";

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

export function genreIcon(name: string): string {
  const key = normalizeGenre(name);
  const exact = GENRE_ICONS[key];
  if (exact) return exact;
  for (const [pattern, icon] of GENRE_ICON_PATTERNS) {
    if (key.includes(pattern)) return icon;
  }
  return DEFAULT_GENRE_ICON;
}

function buildArtUrl(name: string): string {
  const key = normalizeGenre(name);
  const cached = artUrlCache.get(key);
  if (cached) return cached;

  const [bg, accent, mid] = genrePalette(name);
  const hash = hashString(key);
  const cx = 0.2 + ((hash >> 5) % 50) / 100;
  const cy = 0.15 + ((hash >> 11) % 40) / 100;
  const svg =
    `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 64 64'>` +
    `<defs><linearGradient id='g' x1='0' y1='0' x2='1' y2='1'>` +
    `<stop offset='0' stop-color='${bg}'/><stop offset='1' stop-color='${mid}'/>` +
    `</linearGradient>` +
    `<radialGradient id='r' cx='${cx.toFixed(2)}' cy='${cy.toFixed(2)}' r='0.95'>` +
    `<stop offset='0' stop-color='${accent}' stop-opacity='0.55'/>` +
    `<stop offset='1' stop-color='${accent}' stop-opacity='0'/>` +
    `</radialGradient></defs>` +
    `<rect width='64' height='64' fill='url(#g)'/>` +
    `<rect width='64' height='64' fill='url(#r)'/>` +
    `</svg>`;
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
