// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createBoundedMap } from "$lib/core/bounded-cache";
import { genrePalette } from "./genre-art";

const ART_URL_CACHE_MAX_ENTRIES = 1000;

const artUrlCache = createBoundedMap<string, string>(ART_URL_CACHE_MAX_ENTRIES);

function hashString(value: string): number {
  let hash = 2166136261;
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}

function buildCoverArtUrl(seed: string, paletteKey: string): string {
  const normalizedSeed = seed.trim() || "unknown";
  const cacheKey = `${normalizedSeed}\0${paletteKey.trim() || normalizedSeed}`;
  const cached = artUrlCache.get(cacheKey);
  if (cached) return cached;

  const [bg, accent, mid] = genrePalette(paletteKey.trim() || normalizedSeed);
  const hash = hashString(normalizedSeed);
  const cells = 10;
  const half = Math.ceil(cells / 2);
  const last = cells - 1;
  const parts: string[] = [];

  for (let y = 0; y < cells; y++) {
    for (let x = 0; x < half; x++) {
      const bit = (hash >> ((x + y * half) % 32)) & 1;
      const edge =
        (x === 0 && (y === 0 || y === last)) ||
        (x === half - 1 && y === Math.floor(cells / 2));
      const fill = edge ? accent : bit ? mid : bg;
      parts.push(
        `<rect x='${x}' y='${y}' width='1' height='1' fill='${fill}'/>`,
      );
      const mirrorX = last - x;
      if (mirrorX !== x) {
        parts.push(
          `<rect x='${mirrorX}' y='${y}' width='1' height='1' fill='${fill}'/>`,
        );
      }
    }
  }

  const svg = `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 ${cells} ${cells}' shape-rendering='crispEdges'>${parts.join("")}</svg>`;
  const url = `data:image/svg+xml,${encodeURIComponent(svg)}`;
  artUrlCache.set(cacheKey, url);
  return url;
}

export function coverArtFallbackUrl(seed: string, paletteKey?: string): string {
  return buildCoverArtUrl(seed, paletteKey ?? seed);
}

export function trackCoverSeed(track: {
  id: string;
  albumId?: string;
}): string {
  return track.albumId ?? track.id;
}

export function trackCoverPaletteKey(track: {
  artist?: string;
  album?: string;
  title?: string;
}): string {
  return track.artist ?? track.album ?? track.title ?? "track";
}

export function albumCoverSeed(album: { id: string }): string {
  return album.id;
}

export function albumCoverPaletteKey(album: {
  artist?: string;
  name?: string;
}): string {
  return album.artist ?? album.name ?? "album";
}

export function artistCoverSeed(artist: { id: string }): string {
  return artist.id;
}

export function artistCoverPaletteKey(artist: { name?: string }): string {
  return artist.name ?? "artist";
}
